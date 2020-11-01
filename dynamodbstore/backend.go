package dynamodbstore

import (
	"encoding"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/pkg/errors"

	"github.com/ccbrown/keyvaluestore"
)

type Backend struct {
	Client                         BackendClient
	TableName                      string
	AllowEventuallyConsistentReads bool
}

func (b *Backend) WithProfiler(profiler Profiler) *Backend {
	ret := *b
	ret.Client = &ProfilingBackendClient{
		Client:   b.Client,
		Profiler: profiler,
	}
	return &ret
}

func (b *Backend) WithEventuallyConsistentReads() *Backend {
	ret := *b
	ret.AllowEventuallyConsistentReads = true
	return &ret
}

func (b *Backend) AtomicWrite() keyvaluestore.AtomicWriteOperation {
	return &AtomicWriteOperation{
		Backend: b,
	}
}

func (b *Backend) Batch() keyvaluestore.BatchOperation {
	return &BatchOperation{
		FallbackBatchOperation: &keyvaluestore.FallbackBatchOperation{
			Backend: b,
		},
		Backend: b,
	}
}

func attributeValue(v interface{}) *dynamodb.AttributeValue {
	switch v := v.(type) {
	case []byte:
		return &dynamodb.AttributeValue{
			B: []byte(v),
		}
	case string:
		return attributeValue([]byte(v))
	case int:
		return attributeValue(int64(v))
	case int64:
		return &dynamodb.AttributeValue{
			N: aws.String(strconv.FormatInt(v, 10)),
		}
	case encoding.BinaryMarshaler:
		b, err := v.MarshalBinary()
		if err != nil {
			panic("binary marshaler values shouldn't panic. error: " + err.Error())
		}
		return attributeValue(b)
	}
	panic(fmt.Sprintf("unsupported value type: %T", v))
}

func (b *Backend) IncrBy(key string, n int64) (int64, error) {
	result, err := b.Client.UpdateItem(&dynamodb.UpdateItemInput{
		Key:              compositeKey(key, "_"),
		TableName:        aws.String(b.TableName),
		UpdateExpression: aws.String("ADD v :n"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":n": attributeValue(n),
		},
		ReturnValues: aws.String(dynamodb.ReturnValueAllNew),
	})
	if err != nil {
		return 0, errors.Wrap(err, "dynamodb update item request error")
	}
	if v := result.Attributes["v"].N; v != nil {
		return strconv.ParseInt(*v, 10, 64)
	}
	return 0, fmt.Errorf("update item output is missing updated value")
}

func (b *Backend) Delete(key string) (bool, error) {
	result, err := b.Client.DeleteItem(&dynamodb.DeleteItemInput{
		Key:          compositeKey(key, "_"),
		TableName:    aws.String(b.TableName),
		ReturnValues: aws.String(dynamodb.ReturnValueAllOld),
	})
	if err != nil {
		return false, errors.Wrap(err, "dynamodb delete item request error")
	}
	return result.Attributes != nil, nil
}

func attributeStringValue(v *dynamodb.AttributeValue) *string {
	if v != nil {
		switch {
		case v.B != nil:
			s := string(v.B)
			return &s
		case v.N != nil:
			return v.N
		}
	}
	return nil
}

func attributeStringSliceValue(v *dynamodb.AttributeValue) []string {
	if v == nil {
		return nil
	}
	bs := v.BS
	if len(bs) == 0 {
		return nil
	}
	members := make([]string, len(bs))
	for i, v := range bs {
		members[i] = string(v)
	}
	return members
}

func (b *Backend) Get(key string) (*string, error) {
	result, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:            compositeKey(key, "_"),
		TableName:      aws.String(b.TableName),
		ConsistentRead: aws.Bool(!b.AllowEventuallyConsistentReads),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get item request error")
	}
	if result.Item == nil || result.Item["v"] == nil {
		return nil, nil
	}
	return attributeStringValue(result.Item["v"]), nil
}

func compositeKey(hash, sort string) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"hk": &dynamodb.AttributeValue{
			B: []byte(hash),
		},
		"rk": &dynamodb.AttributeValue{
			B: []byte(sort),
		},
	}
}

func newItem(key, sort string, attrs map[string]*dynamodb.AttributeValue) map[string]*dynamodb.AttributeValue {
	item := compositeKey(key, sort)
	for name, attr := range attrs {
		item[name] = attr
	}
	return item
}

func (b *Backend) Set(key string, value interface{}) error {
	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(b.TableName),
		Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
			"v": attributeValue(value),
		}),
	}); err != nil {
		return errors.Wrap(err, "dynamodb put item request error")
	}
	return nil
}

func (b *Backend) SetNX(key string, value interface{}) (bool, error) {
	return b.setNX(key, "_", map[string]*dynamodb.AttributeValue{"v": attributeValue(value)})
}

func (b *Backend) setNX(key string, sortKey string, valueMap map[string]*dynamodb.AttributeValue) (bool, error) {
	var conditions []string

	for k := range valueMap {
		conditions = append(conditions, fmt.Sprintf("attribute_not_exists(%s)", k))
	}

	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(b.TableName),
		Item:                newItem(key, sortKey, valueMap),
		ConditionExpression: aws.String(strings.Join(conditions, " and ")),
	}); err != nil {
		if err := err.(awserr.Error); err != nil && err.Code() == "ConditionalCheckFailedException" {
			return false, nil
		}
		return false, errors.Wrap(err, "dynamodb put item request error")
	}
	return true, nil
}

func (b *Backend) SetXX(key string, value interface{}) (bool, error) {
	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(b.TableName),
		Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
			"v": attributeValue(value),
		}),
		ConditionExpression: aws.String("attribute_exists(v)"),
	}); err != nil {
		if err := err.(awserr.Error); err != nil && err.Code() == "ConditionalCheckFailedException" {
			return false, nil
		}
		return false, errors.Wrap(err, "dynamodb put item request error")
	}
	return true, nil
}

func (b *Backend) SetEQ(key string, value, oldValue interface{}) (bool, error) {
	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(b.TableName),
		Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
			"v": attributeValue(value),
		}),
		ConditionExpression: aws.String("v = :v"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": attributeValue(oldValue),
		},
	}); err != nil {
		if err := err.(awserr.Error); err != nil && err.Code() == "ConditionalCheckFailedException" {
			return false, nil
		}
		return false, errors.Wrap(err, "dynamodb put item request error")
	}
	return true, nil
}

func serializeSMembers(member interface{}, members ...interface{}) [][]byte {
	bs := make([][]byte, 1+len(members))
	bs[0] = []byte(*keyvaluestore.ToString(member))
	for i, member := range members {
		bs[i+1] = []byte(*keyvaluestore.ToString(member))
	}
	return bs
}

func (b *Backend) SAdd(key string, member interface{}, members ...interface{}) error {
	if _, err := b.Client.UpdateItem(&dynamodb.UpdateItemInput{
		Key:              compositeKey(key, "_"),
		TableName:        aws.String(b.TableName),
		UpdateExpression: aws.String("ADD v :v"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": &dynamodb.AttributeValue{
				BS: serializeSMembers(member, members...),
			},
		},
	}); err != nil {
		return errors.Wrap(err, "dynamodb update item request error")
	}
	return nil
}

func (b *Backend) SRem(key string, member interface{}, members ...interface{}) error {
	if _, err := b.Client.UpdateItem(&dynamodb.UpdateItemInput{
		Key:              compositeKey(key, "_"),
		TableName:        aws.String(b.TableName),
		UpdateExpression: aws.String("DELETE v :v"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": &dynamodb.AttributeValue{
				BS: serializeSMembers(member, members...),
			},
		},
	}); err != nil {
		return errors.Wrap(err, "dynamodb update item request error")
	}
	return nil
}

func (b *Backend) SMembers(key string) ([]string, error) {
	result, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:            compositeKey(key, "_"),
		TableName:      aws.String(b.TableName),
		ConsistentRead: aws.Bool(!b.AllowEventuallyConsistentReads),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get item request error")
	}
	if result.Item == nil || result.Item["v"] == nil {
		return nil, nil
	}
	return attributeStringSliceValue(result.Item["v"]), nil
}

func encodeHashFieldName(name string) string {
	return "~" + base64.RawURLEncoding.EncodeToString([]byte(name))
}

func decodeHashFieldName(name string) string {
	if !strings.HasPrefix(name, "~") {
		return ""
	}
	b, _ := base64.RawURLEncoding.DecodeString(name[1:])
	return string(b)
}

func (b *Backend) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	assignments := make([]string, 0, 1+len(fields))
	names := make(map[string]*string, 1+len(fields))
	values := make(map[string]*dynamodb.AttributeValue, 1+len(fields))
	assignments = append(assignments, "#n0 = :v0")
	names["#n0"] = aws.String(encodeHashFieldName(field))
	values[":v0"] = &dynamodb.AttributeValue{
		B: []byte(*keyvaluestore.ToString(value)),
	}
	for i, field := range fields {
		namePlaceholder := "#n" + strconv.Itoa(i+1)
		valuePlaceholder := ":v" + strconv.Itoa(i+1)
		assignments = append(assignments, namePlaceholder+" = "+valuePlaceholder)
		names[namePlaceholder] = aws.String(encodeHashFieldName(field.Key))
		values[valuePlaceholder] = &dynamodb.AttributeValue{
			B: []byte(*keyvaluestore.ToString(field.Value)),
		}
	}
	if _, err := b.Client.UpdateItem(&dynamodb.UpdateItemInput{
		Key:                       compositeKey(key, "_"),
		TableName:                 aws.String(b.TableName),
		UpdateExpression:          aws.String("SET " + strings.Join(assignments, ", ")),
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
	}); err != nil {
		return errors.Wrap(err, "dynamodb update item request error")
	}
	return nil
}

func (b *Backend) HDel(key, field string, fields ...string) error {
	placeholders := make([]string, 0, 1+len(fields))
	names := make(map[string]*string, 1+len(fields))
	placeholders = append(placeholders, "#n0")
	names["#n0"] = aws.String(encodeHashFieldName(field))
	for i, field := range fields {
		placeholder := "#n" + strconv.Itoa(i+1)
		placeholders = append(placeholders, placeholder)
		names[placeholder] = aws.String(encodeHashFieldName(field))
	}
	if _, err := b.Client.UpdateItem(&dynamodb.UpdateItemInput{
		Key:                      compositeKey(key, "_"),
		TableName:                aws.String(b.TableName),
		UpdateExpression:         aws.String("REMOVE " + strings.Join(placeholders, ", ")),
		ExpressionAttributeNames: names,
	}); err != nil {
		return errors.Wrap(err, "dynamodb update item request error")
	}
	return nil
}

func (b *Backend) HGet(key, field string) (*string, error) {
	attributeName := encodeHashFieldName(field)
	result, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:                  compositeKey(key, "_"),
		TableName:            aws.String(b.TableName),
		ProjectionExpression: aws.String("#n"),
		ExpressionAttributeNames: map[string]*string{
			"#n": &attributeName,
		},
		ConsistentRead: aws.Bool(!b.AllowEventuallyConsistentReads),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get item request error")
	}
	if result.Item == nil || result.Item[attributeName] == nil {
		return nil, nil
	}
	return attributeStringValue(result.Item[attributeName]), nil
}

func (b *Backend) HGetAll(key string) (map[string]string, error) {
	result, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:            compositeKey(key, "_"),
		TableName:      aws.String(b.TableName),
		ConsistentRead: aws.Bool(!b.AllowEventuallyConsistentReads),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get item request error")
	}
	if result.Item == nil {
		return nil, nil
	}
	ret := make(map[string]string, len(result.Item))
	for k, v := range result.Item {
		if name := decodeHashFieldName(k); name != "" {
			if v := attributeStringValue(v); v != nil {
				ret[name] = *v
			}
		}
	}
	return ret, nil
}

const floatSortKeyNumBytes = 8

func floatSortKey(f float64) string {
	n := math.Float64bits(f)
	if (n & (1 << 63)) != 0 {
		n ^= 0xffffffffffffffff
	} else {
		n ^= 0x8000000000000000
	}
	buf := make([]byte, floatSortKeyNumBytes)
	binary.BigEndian.PutUint64(buf, n)
	return string(buf)
}

func sortKeyFloat(key string) float64 {
	if len(key) < floatSortKeyNumBytes {
		return 0
	}
	n := binary.BigEndian.Uint64([]byte(key))
	if (n & (1 << 63)) == 0 {
		n ^= 0xffffffffffffffff
	} else {
		n ^= 0x8000000000000000
	}
	return math.Float64frombits(n)
}

func floatSortKeyAfter(f float64) string {
	n := math.Float64bits(f)
	if (n & (1 << 63)) != 0 {
		n ^= 0xffffffffffffffff
	} else {
		n ^= 0x8000000000000000
	}
	n++
	if n == 0 {
		return ""
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	return string(buf)
}

func (b *Backend) ZAdd(key string, member interface{}, score float64) error {
	s := *keyvaluestore.ToString(member)
	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(b.TableName),
		Item: newItem(key, s, map[string]*dynamodb.AttributeValue{
			"v":   attributeValue(s),
			"rk2": attributeValue(floatSortKey(score) + s),
		}),
	}); err != nil {
		return errors.Wrap(err, "dynamodb put item request error")
	}
	return nil
}

func (b *Backend) ZHAdd(key, field string, member interface{}) error {
	s := *keyvaluestore.ToString(member)
	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(b.TableName),
		Item: newItem(key, field, map[string]*dynamodb.AttributeValue{
			"v":   attributeValue(s),
			"rk2": attributeValue(floatSortKey(0.0) + field),
		}),
	}); err != nil {
		return errors.Wrap(err, "dynamodb put item request error")
	}
	return nil
}

func (b *Backend) ZScore(key string, member interface{}) (*float64, error) {
	s := *keyvaluestore.ToString(member)
	result, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:            compositeKey(key, s),
		TableName:      aws.String(b.TableName),
		ConsistentRead: aws.Bool(!b.AllowEventuallyConsistentReads),
	})
	if err != nil {
		return nil, errors.Wrap(err, "dynamodb get item request error")
	}
	if result.Item != nil {
		if rk2 := attributeStringValue(result.Item["rk2"]); rk2 != nil {
			score := sortKeyFloat(*rk2)
			return &score, nil
		}
	}
	return nil, nil
}

func (b *Backend) ZIncrBy(key string, member string, n float64) (float64, error) {
	var retValue float64

	err := runContentiousMethod(func() (bool, error) {
		var newValue float64

		s := *keyvaluestore.ToString(member)

		success, err := b.checkAndSet(key, s, "rk2", func(prev *string) (interface{}, error) {
			if prev != nil {
				floatValue := sortKeyFloat(*prev)
				newValue = floatValue + n
			} else {
				newValue = n
			}

			return floatSortKey(newValue) + s, nil
		}, map[string]interface{}{"v": s})

		if err != nil {
			return false, err
		} else if !success {
			return false, fmt.Errorf("unable to increment due to contention")
		}

		retValue = newValue
		return true, nil
	})

	if err != nil {
		return 0, err
	}

	return retValue, nil
}

func (b *Backend) ZRem(key string, member interface{}) error {
	s := *keyvaluestore.ToString(member)
	return b.ZHRem(key, s)
}

func (b *Backend) ZHRem(key, field string) error {
	if _, err := b.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: aws.String(b.TableName),
		Key:       compositeKey(key, field),
	}); err != nil {
		return errors.Wrap(err, "dynamodb delete item request error")
	}
	return nil
}

func minMaxFloatSortKeys(min, max float64) (string, string) {
	minSortKey := "[" + floatSortKey(min)
	if min == math.Inf(-1) {
		minSortKey = "-"
	}
	maxSortKey := "(" + floatSortKeyAfter(max)
	if maxSortKey == "(" {
		maxSortKey = "+"
	}
	return minSortKey, maxSortKey
}

func (b *Backend) ZCount(key string, min, max float64) (int, error) {
	minSortKey, maxSortKey := minMaxFloatSortKeys(min, max)
	return b.zCount(key, minSortKey, maxSortKey, true)
}

func (b *Backend) ZLexCount(key, min, max string) (int, error) {
	return b.zCount(key, min, max, false)
}

func (b *Backend) zCount(key string, min, max string, secondaryIndex bool) (int, error) {
	if (min[0] == '(' && max[0] != '+') || (max[0] == '(' && min[0] != '-') {
		// There's no way to represent ranges with exclusive bounds as a DynamoDB condition (BETWEEN
		// is inclusive only). Instead, we have to do two queries.
		inOrAfterCount, err := b.zCount(key, min, "+", secondaryIndex)
		if err != nil {
			return 0, err
		}
		maxOpp := "[" + max[1:]
		if maxOpp[0] == '[' {
			maxOpp = "(" + max[1:]
		}
		afterCount, err := b.zCount(key, maxOpp, "+", secondaryIndex)
		if err != nil {
			return 0, err
		}
		if afterCount >= inOrAfterCount {
			return 0, nil
		}
		return inOrAfterCount - afterCount, nil
	}

	condition, attributeValues := queryCondition(key, min, max, secondaryIndex)
	if condition == "" {
		return 0, nil
	}
	input := &dynamodb.QueryInput{
		TableName:                 aws.String(b.TableName),
		ConsistentRead:            aws.Bool(!b.AllowEventuallyConsistentReads),
		KeyConditionExpression:    aws.String(condition),
		ExpressionAttributeValues: attributeValues,
		Select:                    aws.String(dynamodb.SelectCount),
	}
	if secondaryIndex {
		input.IndexName = aws.String("rk2")
	}

	count := 0
	for {
		result, err := b.Client.Query(input)
		if err != nil {
			return 0, errors.Wrap(err, "dynamodb query request error")
		}
		if result.Count == nil {
			return 0, fmt.Errorf("no count returned by dynamodb query")
		}
		count += int(*result.Count)
		if len(result.LastEvaluatedKey) == 0 {
			return count, nil
		}
		input.ExclusiveStartKey = result.LastEvaluatedKey
	}
}

func (b *Backend) ZRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.zRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) zRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	minSortKey, maxSortKey := minMaxFloatSortKeys(min, max)
	return b.zRangeByLex(key, minSortKey, maxSortKey, limit, false, true)
}

func (b *Backend) ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.zRevRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zRevRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) zRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	minSortKey, maxSortKey := minMaxFloatSortKeys(min, max)
	return b.zRangeByLex(key, minSortKey, maxSortKey, limit, true, true)
}

func (b *Backend) ZRangeByLex(key string, min, max string, limit int) ([]string, error) {
	members, err := b.zRangeByLex(key, min, max, limit, false, false)
	return members.Values(), err
}

func (b *Backend) ZHRangeByLex(key string, min, max string, limit int) ([]string, error) {
	members, err := b.zRangeByLex(key, min, max, limit, false, false)
	return members.Values(), err
}

func (b *Backend) ZRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	members, err := b.zRangeByLex(key, min, max, limit, true, false)
	return members.Values(), err
}

func (b *Backend) ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	members, err := b.zRangeByLex(key, min, max, limit, true, false)
	return members.Values(), err
}

func queryCondition(key, min, max string, secondaryIndex bool) (string, map[string]*dynamodb.AttributeValue) {
	minSort := min[1:]
	maxSort := max[1:]

	attributeValues := map[string]*dynamodb.AttributeValue{
		":hash": attributeValue(key),
	}
	if min != "-" {
		attributeValues[":minSort"] = attributeValue(minSort)
	}
	if max != "+" {
		attributeValues[":maxSort"] = attributeValue(maxSort)
	}

	rangeKey := "rk"
	if secondaryIndex {
		rangeKey = "rk2"
	}

	condition := "hk = :hash AND " + rangeKey + " BETWEEN :minSort AND :maxSort"
	if min == "-" && max == "+" {
		condition = "hk = :hash"
	} else if min == "-" {
		condition = "hk = :hash AND " + rangeKey + " <= :maxSort"
	} else if max == "+" {
		condition = "hk = :hash AND " + rangeKey + " >= :minSort"
	} else if minSort > maxSort {
		return "", nil
	}

	return condition, attributeValues
}

func (b *Backend) zRangeByLex(key, min, max string, limit int, reverse, secondaryIndex bool) (members keyvaluestore.ScoredMembers, err error) {
	var startKey map[string]*dynamodb.AttributeValue

	condition, attributeValues := queryCondition(key, min, max, secondaryIndex)
	if condition == "" {
		return nil, nil
	}

	rangeKey := "rk"
	if secondaryIndex {
		rangeKey = "rk2"
	}

	for limit == 0 || len(members) < limit {
		input := &dynamodb.QueryInput{
			TableName:                 aws.String(b.TableName),
			ConsistentRead:            aws.Bool(!b.AllowEventuallyConsistentReads),
			KeyConditionExpression:    aws.String(condition),
			ExpressionAttributeValues: attributeValues,
			ExclusiveStartKey:         startKey,
			ScanIndexForward:          aws.Bool(!reverse),
		}
		if secondaryIndex {
			input.IndexName = aws.String("rk2")
		}
		if limit > 0 {
			input.Limit = aws.Int64(int64(limit - len(members)))
		}
		result, err := b.Client.Query(input)
		if err != nil {
			return nil, errors.Wrap(err, "dynamodb query request error")
		}
		for _, item := range result.Items {
			sort := *attributeStringValue(item[rangeKey])
			if (min[0] == '(' && sort == min[1:]) || (max[0] == '(' && sort == max[1:]) {
				continue
			}

			var score float64

			if v, ok := item["rk2"]; ok {
				score = sortKeyFloat(*attributeStringValue(v))
			}

			members = append(members, &keyvaluestore.ScoredMember{
				Score: score,
				Value: *attributeStringValue(item["v"]),
			})
		}
		if result.LastEvaluatedKey == nil {
			break
		}
		startKey = result.LastEvaluatedKey
	}
	return members, nil
}

func (b *Backend) checkAndSet(key string, sortKey string, attributeToChange string, transform func(prev *string) (interface{}, error), otherValues map[string]interface{}) (bool, error) {
	compKey := compositeKey(key, sortKey)

	getResult, err := b.Client.GetItem(&dynamodb.GetItemInput{
		Key:            compKey,
		TableName:      aws.String(b.TableName),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return false, errors.Wrap(err, "dynamodb get item request error")
	}

	var prev *string
	if getResult.Item != nil {
		prev = attributeStringValue(getResult.Item[attributeToChange])
	}

	newValue, err := transform(prev)
	if err != nil {
		return false, err
	} else if newValue == nil {
		return true, nil
	}

	attributeValues := map[string]*dynamodb.AttributeValue{
		attributeToChange: attributeValue(newValue),
	}

	for k, v := range otherValues {
		attributeValues[k] = attributeValue(v)
	}

	if prev == nil {
		return b.setNX(key, sortKey, attributeValues)
	}

	if _, err := b.Client.PutItem(&dynamodb.PutItemInput{
		TableName:           aws.String(b.TableName),
		Item:                newItem(key, sortKey, attributeValues),
		ConditionExpression: aws.String(fmt.Sprintf("%s = :v", attributeToChange)),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v": getResult.Item[attributeToChange],
		},
	}); err != nil {
		if err := err.(awserr.Error); err != nil && err.Code() == "ConditionalCheckFailedException" {
			return false, nil
		}
		return false, errors.Wrap(err, "dynamodb put item request error")
	}
	return true, nil
}

const contentiousMethodRetries = 3

func runContentiousMethod(f func() (bool, error)) error {
	for i := 0; i < contentiousMethodRetries; i++ {
		success, err := f()
		if err != nil {
			return err
		} else if success {
			return nil
		}
	}
	return fmt.Errorf("unable to run method due to contention, tried %d times", contentiousMethodRetries)
}

func CreateDefaultTable(client *dynamodb.DynamoDB, tableName string) error {
	return createDefaultTable(client, tableName, true)
}

func createDefaultTable(client *dynamodb.DynamoDB, tableName string, tryPayPerRequest bool) error {
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("hk"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeB),
			}, {
				AttributeName: aws.String("rk"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeB),
			}, {
				AttributeName: aws.String("rk2"),
				AttributeType: aws.String(dynamodb.ScalarAttributeTypeB),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("hk"),
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			}, {
				AttributeName: aws.String("rk"),
				KeyType:       aws.String(dynamodb.KeyTypeRange),
			},
		},
		LocalSecondaryIndexes: []*dynamodb.LocalSecondaryIndex{
			{
				IndexName: aws.String("rk2"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("hk"),
						KeyType:       aws.String(dynamodb.KeyTypeHash),
					}, {
						AttributeName: aws.String("rk2"),
						KeyType:       aws.String(dynamodb.KeyTypeRange),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String(dynamodb.ProjectionTypeAll),
				},
			},
		},
		TableName: &tableName,
	}
	if tryPayPerRequest {
		input.BillingMode = aws.String(dynamodb.BillingModePayPerRequest)
	} else {
		input.ProvisionedThroughput = &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		}
	}
	_, err := client.CreateTable(input)
	if err, ok := err.(awserr.Error); ok && err.Code() == "ValidationException" && tryPayPerRequest {
		// Docker DynamoDB doesn't support pay-per-request billing mode.
		return createDefaultTable(client, tableName, false)
	}
	return err
}
