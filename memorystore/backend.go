package memorystore

import (
	"encoding/binary"
	"math"
	"strconv"
	"sync"

	"github.com/ccbrown/go-immutable"

	"github.com/ccbrown/keyvaluestore"
)

type Backend struct {
	m     map[string]interface{}
	mutex sync.Mutex
}

func NewBackend() *Backend {
	return &Backend{
		m: make(map[string]interface{}),
	}
}

// Erases everything in the backend and makes it like-new.
func (b *Backend) Reinitialize() {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.m = make(map[string]interface{})
}

func (b *Backend) Delete(key string) (bool, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.delete(key), nil
}

func (b *Backend) delete(key string) bool {
	_, ok := b.m[key]
	delete(b.m, key)
	return ok
}

func (b *Backend) Batch() keyvaluestore.BatchOperation {
	return &keyvaluestore.FallbackBatchOperation{
		Backend: b,
	}
}

func (b *Backend) AtomicWrite() keyvaluestore.AtomicWriteOperation {
	return &AtomicWriteOperation{
		Backend: b,
	}
}

func (b *Backend) Get(key string) (*string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.get(key), nil
}

func (b *Backend) get(key string) *string {
	if v, ok := b.m[key]; ok {
		return keyvaluestore.ToString(v)
	}
	return nil
}

func (b *Backend) Set(key string, value interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.set(key, value)
	return nil
}

func (b *Backend) set(key string, value interface{}) {
	b.m[key] = value
}

func (b *Backend) IncrBy(key string, n int64) (int64, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.incrBy(key, n)
}

func (b *Backend) incrBy(key string, n int64) (int64, error) {
	if v, ok := b.m[key]; ok {
		if s := keyvaluestore.ToString(v); s != nil {
			i, err := strconv.ParseInt(*s, 10, 64)
			if err != nil {
				return 0, err
			}
			b.m[key] = strconv.FormatInt(i+n, 10)
			return i + n, nil
		}
	}
	b.m[key] = strconv.FormatInt(n, 10)
	return n, nil
}

func (b *Backend) SAdd(key string, member interface{}, members ...interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.sadd(key, member, members...)
	return nil
}

func (b *Backend) sadd(key string, member interface{}, members ...interface{}) {
	s, ok := b.m[key].(map[string]struct{})
	if !ok {
		s = make(map[string]struct{})
	}
	s[*keyvaluestore.ToString(member)] = struct{}{}
	for _, member := range members {
		s[*keyvaluestore.ToString(member)] = struct{}{}
	}
	b.m[key] = s
}

func (b *Backend) SRem(key string, member interface{}, members ...interface{}) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.srem(key, member, members...)
}

func (b *Backend) srem(key string, member interface{}, members ...interface{}) error {
	s, ok := b.m[key].(map[string]struct{})
	if !ok {
		return nil
	}
	delete(s, *keyvaluestore.ToString(member))
	for _, member := range members {
		delete(s, *keyvaluestore.ToString(member))
	}
	if len(s) == 0 {
		delete(b.m, key)
	}
	return nil
}

func (b *Backend) SMembers(key string) ([]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	s, ok := b.m[key].(map[string]struct{})
	if !ok {
		return nil, nil
	}
	var results []string
	for k := range s {
		results = append(results, k)
	}
	return results, nil
}

func (b *Backend) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.hset(key, field, value, fields...)
}

func (b *Backend) hset(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	h, ok := b.m[key].(map[string]string)
	if !ok {
		h = make(map[string]string)
	}
	h[field] = *keyvaluestore.ToString(value)
	for _, field := range fields {
		h[field.Key] = *keyvaluestore.ToString(field.Value)
	}
	b.m[key] = h
	return nil
}

func (b *Backend) HDel(key string, field string, fields ...string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.hdel(key, field, fields...)
}

func (b *Backend) hdel(key string, field string, fields ...string) error {
	h, ok := b.m[key].(map[string]string)
	if !ok {
		return nil
	}
	delete(h, field)
	for _, field := range fields {
		delete(h, field)
	}
	if len(h) == 0 {
		delete(b.m, key)
	}
	return nil
}

func (b *Backend) HGet(key, field string) (*string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.hget(key, field), nil
}

func (b *Backend) hget(key, field string) *string {
	if v, ok := b.hgetall(key)[field]; ok {
		return &v
	}
	return nil
}

func (b *Backend) HGetAll(key string) (map[string]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.hgetall(key), nil
}

func (b *Backend) hgetall(key string) map[string]string {
	h, ok := b.m[key].(map[string]string)
	if !ok {
		return nil
	}
	return h
}

func (b *Backend) SetNX(key string, value interface{}) (bool, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, ok := b.m[key]; ok {
		return false, nil
	}

	b.m[key] = value
	return true, nil
}

func (b *Backend) SetXX(key string, value interface{}) (bool, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if _, ok := b.m[key]; !ok {
		return false, nil
	}

	b.m[key] = value
	return true, nil
}

func (b *Backend) SetEQ(key string, value, oldValue interface{}) (bool, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if v, ok := b.m[key]; !ok || *keyvaluestore.ToString(v) != *keyvaluestore.ToString(oldValue) {
		return false, nil
	}

	b.m[key] = value
	return true, nil
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

type sortedSet struct {
	scoresByMember map[string]float64
	m              *immutable.OrderedMap
}

func (b *Backend) zhadd(key, field string, member interface{}, f func(previousScore *float64) (float64, error)) (float64, error) {
	s, _ := b.m[key].(*sortedSet)
	if s == nil {
		s = &sortedSet{
			scoresByMember: make(map[string]float64),
		}
	}

	var previousScore *float64

	if prev, ok := s.scoresByMember[field]; ok {
		s.m = s.m.Delete(floatSortKey(prev) + field)
		previousScore = &prev
	}

	newScore, err := f(previousScore)

	if err != nil {
		return 0, err
	} else {
		v := *keyvaluestore.ToString(member)
		s.m = s.m.Set(floatSortKey(newScore)+field, v)
		s.scoresByMember[field] = newScore
	}

	b.m[key] = s
	return newScore, nil
}

func (b *Backend) ZAdd(key string, member interface{}, score float64) error {
	s := *keyvaluestore.ToString(member)
	return b.ZHAdd(key, s, s, score)
}

func (b *Backend) ZHAdd(key, field string, member interface{}, score float64) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	_, err := b.zhadd(key, field, member, func(previousScore *float64) (float64, error) {
		return score, nil
	})
	return err
}

func (b *Backend) ZScore(key string, member interface{}) (*float64, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if s, _ := b.m[key].(*sortedSet); s != nil {
		v := *keyvaluestore.ToString(member)
		if prev, ok := s.scoresByMember[v]; ok {
			return &prev, nil
		}
	}

	return nil, nil
}

func (b *Backend) ZIncrBy(key string, member string, n float64) (float64, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	s := *keyvaluestore.ToString(member)
	return b.zhadd(key, s, s, func(previousScore *float64) (float64, error) {
		if previousScore != nil {
			return *previousScore + n, nil
		}

		return n, nil
	})
}

func (b *Backend) zscore(key string, member interface{}) *float64 {
	s, _ := b.m[key].(*sortedSet)
	if s != nil {
		v := *keyvaluestore.ToString(member)
		if score, ok := s.scoresByMember[v]; ok {
			return &score
		}
	}
	return nil
}

func (b *Backend) ZRem(key string, member interface{}) error {
	s := *keyvaluestore.ToString(member)
	return b.ZHRem(key, s)
}

func (b *Backend) ZHRem(key, field string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.zhrem(key, field)
}

func (b *Backend) zhrem(key, field string) error {
	s, _ := b.m[key].(*sortedSet)
	if s != nil {
		if previous, ok := s.scoresByMember[field]; ok {
			s.m = s.m.Delete(floatSortKey(previous) + field)
			delete(s.scoresByMember, field)
			b.m[key] = s
		}
	}
	return nil
}

func (b *Backend) ZRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if members, err := b.zRangeByScoreWithScores(key, min, max, limit); err != nil {
		return nil, err
	} else {
		return members.Values(), nil
	}
}

func (b *Backend) ZHRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return b.ZRangeByScore(key, min, max, limit)
}

func (b *Backend) ZRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.zRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) ZHRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.ZRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) zRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	s, _ := b.m[key].(*sortedSet)
	if s == nil {
		return nil, nil
	}

	var results []*keyvaluestore.ScoredMember

	minSortKey := floatSortKey(min)
	maxSortKeyPrefix := floatSortKey(max)

	next := s.m.MaxBefore(minSortKey)
	if next == nil {
		next = s.m.Min()
	} else {
		next = next.Next()
	}

	for (limit == 0 || len(results) < limit) && next != nil && next.Key().(string)[:len(maxSortKeyPrefix)] <= maxSortKeyPrefix {
		results = append(results, &keyvaluestore.ScoredMember{
			Score: sortKeyFloat(next.Key().(string)),
			Value: next.Value().(string),
		})
		next = next.Next()
	}

	return results, nil
}

func (b *Backend) ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if members, err := b.zRevRangeByScoreWithScores(key, min, max, limit); err != nil {
		return nil, err
	} else {
		return members.Values(), nil
	}
}

func (b *Backend) ZHRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return b.ZRevRangeByScore(key, min, max, limit)
}

func (b *Backend) ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.zRevRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) ZHRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.ZRevRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) zRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	s, _ := b.m[key].(*sortedSet)
	if s == nil {
		return nil, nil
	}

	var results []*keyvaluestore.ScoredMember

	minSortKey := floatSortKey(min)
	sortKeyAfterMax := floatSortKeyAfter(max)

	var next *immutable.OrderedMapElement
	if sortKeyAfterMax == "" {
		next = s.m.Max()
	} else {
		next = s.m.MaxBefore(sortKeyAfterMax)
	}

	for (limit == 0 || len(results) < limit) && next != nil && next.Key().(string) >= minSortKey {
		results = append(results, &keyvaluestore.ScoredMember{
			Score: sortKeyFloat(next.Key().(string)),
			Value: next.Value().(string),
		})
		next = next.Prev()
	}

	return results, nil
}

func (b *Backend) ZCount(key string, min, max float64) (int, error) {
	members, err := b.ZRangeByScore(key, min, max, 0)
	return len(members), err
}

func (b *Backend) ZLexCount(key string, min, max string) (int, error) {
	members, err := b.ZRangeByLex(key, min, max, 0)
	return len(members), err
}

func (b *Backend) ZRangeByLex(key string, min, max string, limit int) ([]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	s, _ := b.m[key].(*sortedSet)
	if s == nil {
		return nil, nil
	}

	var results []string

	sortKeyPrefix := string(floatSortKey(0.0))

	var next *immutable.OrderedMapElement
	if min == "-" {
		next = s.m.Min()
	} else {
		next = s.m.MinAfter(sortKeyPrefix + min[1:])
		if min[0] == '[' {
			if next == nil {
				if x := s.m.Max(); x != nil && x.Key().(string)[len(sortKeyPrefix):] == min[1:] {
					next = x
				}
			} else if x := next.Prev(); x != nil && x.Key().(string)[len(sortKeyPrefix):] == min[1:] {
				next = x
			}
		}
	}

	for (limit == 0 || len(results) < limit) && next != nil {
		lex := next.Key().(string)[len(sortKeyPrefix):]
		if max != "+" && (lex > max[1:] || (max[0] == '(' && lex == max[1:])) {
			break
		}
		results = append(results, next.Value().(string))
		next = next.Next()
	}

	return results, nil
}

func (b *Backend) ZHRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.ZRangeByLex(key, min, max, limit)
}

func (b *Backend) ZRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	s, _ := b.m[key].(*sortedSet)
	if s == nil {
		return nil, nil
	}

	var results []string

	sortKeyPrefix := string(floatSortKey(0.0))

	var next *immutable.OrderedMapElement
	if max == "+" {
		next = s.m.Max()
	} else {
		next = s.m.MaxBefore(sortKeyPrefix + max[1:])
		if max[0] == '[' {
			if next == nil {
				if x := s.m.Min(); x != nil && x.Key().(string)[len(sortKeyPrefix):] == min[1:] {
					next = x
				}
			} else if x := next.Next(); x != nil && x.Key().(string)[len(sortKeyPrefix):] == max[1:] {
				next = x
			}
		}
	}

	for (limit == 0 || len(results) < limit) && next != nil {
		lex := next.Key().(string)[len(sortKeyPrefix):]
		if min != "-" && (lex < min[1:] || (min[0] == '(' && lex == min[1:])) {
			break
		}
		results = append(results, next.Value().(string))
		next = next.Prev()
	}

	return results, nil
}

func (b *Backend) ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.ZRevRangeByLex(key, min, max, limit)
}
