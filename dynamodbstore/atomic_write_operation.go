package dynamodbstore

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/pkg/errors"

	"github.com/ccbrown/keyvaluestore"
)

type AtomicWriteOperation struct {
	Backend *Backend

	items   []*dynamodb.TransactWriteItem
	results []*atomicWriteResult
}

type atomicWriteResult struct {
	cancellationReason *dynamodb.CancellationReason
}

func (r *atomicWriteResult) ConditionalFailed() bool {
	return r.cancellationReason != nil && r.cancellationReason.Code != nil && *r.cancellationReason.Code == "ConditionalCheckFailed"
}

func (op *AtomicWriteOperation) write(item dynamodb.TransactWriteItem) *atomicWriteResult {
	op.items = append(op.items, &item)
	ret := &atomicWriteResult{}
	op.results = append(op.results, ret)
	return ret
}

func (op *AtomicWriteOperation) Set(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
				"v": attributeValue(value),
			}),
			TableName: &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			ConditionExpression: aws.String("attribute_not_exists(v)"),
			Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
				"v": attributeValue(value),
			}),
			TableName: &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) SetXX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			ConditionExpression: aws.String("attribute_exists(v)"),
			Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
				"v": attributeValue(value),
			}),
			TableName: &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			ConditionExpression: aws.String("v = :v"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v": attributeValue(oldValue),
			},
			Item: newItem(key, "_", map[string]*dynamodb.AttributeValue{
				"v": attributeValue(value),
			}),
			TableName: &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Delete: &dynamodb.Delete{
			Key:       compositeKey(key, "_"),
			TableName: &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) DeleteXX(key string) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Delete: &dynamodb.Delete{
			ConditionExpression: aws.String("attribute_exists(v)"),
			Key:                 compositeKey(key, "_"),
			TableName:           &op.Backend.TableName,
		},
	})
}

func (op *AtomicWriteOperation) IncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Update: &dynamodb.Update{
			Key:              compositeKey(key, "_"),
			TableName:        &op.Backend.TableName,
			UpdateExpression: aws.String("ADD v :n"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":n": attributeValue(n),
			},
		},
	})
}

func (op *AtomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	s := *keyvaluestore.ToString(member)
	return op.write(dynamodb.TransactWriteItem{
		Put: &dynamodb.Put{
			TableName: &op.Backend.TableName,
			Item: newItem(key, s, map[string]*dynamodb.AttributeValue{
				"v":   attributeValue(s),
				"rk2": attributeValue(floatSortKey(score) + s),
			}),
		},
	})
}

func (op *AtomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	s := *keyvaluestore.ToString(member)
	return op.write(dynamodb.TransactWriteItem{
		Delete: &dynamodb.Delete{
			TableName: &op.Backend.TableName,
			Key:       compositeKey(key, s),
		},
	})
}

func (op *AtomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Update: &dynamodb.Update{
			Key:              compositeKey(key, "_"),
			TableName:        &op.Backend.TableName,
			UpdateExpression: aws.String("ADD v :v"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v": &dynamodb.AttributeValue{
					BS: serializeSMembers(member, members...),
				},
			},
		},
	})
}

func (op *AtomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(dynamodb.TransactWriteItem{
		Update: &dynamodb.Update{
			Key:              compositeKey(key, "_"),
			TableName:        &op.Backend.TableName,
			UpdateExpression: aws.String("DELETE v :v"),
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":v": &dynamodb.AttributeValue{
					BS: serializeSMembers(member, members...),
				},
			},
		},
	})
}

func (op *AtomicWriteOperation) Exec() (bool, error) {
	token := make([]byte, 20)
	if _, err := rand.Read(token); err != nil {
		return false, errors.Wrap(err, "unable to generate request token")
	}

	input := &dynamodb.TransactWriteItemsInput{
		TransactItems:      op.items,
		ClientRequestToken: aws.String(base64.RawURLEncoding.EncodeToString(token)),
	}

	attempts := 0
	for {
		_, err := op.Backend.Client.TransactWriteItems(input)
		if err == nil {
			return true, nil
		}

		if err, ok := err.(awserr.Error); ok && err.Code() == "InternalServerError" && attempts < 3 {
			// Internal errors tend to happen if the database was recently recreated. We should
			// retry the request a few times.
			attempts++
			time.Sleep(time.Duration(attempts*attempts) * 100 * time.Millisecond)
			continue
		}

		switch err := err.(type) {
		case *dynamodb.TransactionCanceledException:
			hasErr := false
			hasConditionalCheckFailed := false

			for i, reason := range err.CancellationReasons {
				op.results[i].cancellationReason = reason
				if reason != nil && reason.Code != nil {
					if *reason.Code == "ConditionalCheckFailed" {
						hasConditionalCheckFailed = true
					} else if *reason.Code != "None" {
						hasErr = true
					}
				}
			}

			if hasErr || !hasConditionalCheckFailed {
				return false, err
			}

			return false, nil
		default:
			return false, err
		}
	}
}
