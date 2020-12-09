package redisstore

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis"

	"github.com/ccbrown/keyvaluestore"
)

type Backend struct {
	Client *redis.Client
}

func (b *Backend) Batch() keyvaluestore.BatchOperation {
	return &BatchOperation{
		b.Client.Pipeline(),
	}
}

func (b *Backend) AtomicWrite() keyvaluestore.AtomicWriteOperation {
	return &AtomicWriteOperation{
		Client: b.Client,
	}
}

func (b *Backend) Delete(key string) (bool, error) {
	result := b.Client.Del(key)
	return result.Val() > 0, result.Err()
}

func (b *Backend) Get(key string) (*string, error) {
	v, err := b.Client.Get(key).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &v, err
}

func (b *Backend) Set(key string, value interface{}) error {
	return b.Client.Set(key, value, 0).Err()
}

func (b *Backend) IncrBy(key string, n int64) (int64, error) {
	return b.Client.IncrBy(key, n).Result()
}

func (b *Backend) ZIncrBy(key string, member string, n float64) (float64, error) {
	return b.Client.ZIncrBy(key, n, member).Result()
}

func (b *Backend) SAdd(key string, member interface{}, members ...interface{}) error {
	return b.Client.SAdd(key, append([]interface{}{member}, members...)...).Err()
}

func (b *Backend) SRem(key string, member interface{}, members ...interface{}) error {
	return b.Client.SRem(key, append([]interface{}{member}, members...)...).Err()
}

func (b *Backend) SMembers(key string) ([]string, error) {
	return b.Client.SMembers(key).Result()
}

func (b *Backend) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	m := make(map[string]interface{}, len(fields)+1)
	m[field] = value
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return b.Client.HMSet(key, m).Err()
}

func (b *Backend) HDel(key string, field string, fields ...string) error {
	args := make([]string, 0, len(fields)+1)
	args = append(append(args, field), fields...)
	return b.Client.HDel(key, args...).Err()
}

func (b *Backend) HGet(key, field string) (*string, error) {
	v, err := b.Client.HGet(key, field).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &v, err
}

func (b *Backend) HGetAll(key string) (map[string]string, error) {
	return b.Client.HGetAll(key).Result()
}

func (b *Backend) SetNX(key string, value interface{}) (bool, error) {
	return b.Client.SetNX(key, value, 0).Result()
}

func (b *Backend) SetXX(key string, value interface{}) (bool, error) {
	return b.Client.SetXX(key, value, 0).Result()
}

func (b *Backend) SetEQ(key string, value, oldValue interface{}) (bool, error) {
	err := b.Client.Watch(func(tx *redis.Tx) error {
		if before, err := b.Get(key); err != nil {
			return err
		} else if before == nil || *before != *keyvaluestore.ToString(oldValue) {
			return redis.TxFailedErr
		}

		_, err := tx.TxPipelined(func(pipe redis.Pipeliner) error {
			return pipe.Set(key, value, 0).Err()
		})
		return err
	}, key)
	if err == redis.TxFailedErr {
		return false, nil
	}
	return err == nil, err
}

func (b *Backend) ZAdd(key string, member interface{}, score float64) error {
	return b.Client.ZAdd(key, redis.Z{
		Member: member,
		Score:  score,
	}).Err()
}

func zhHashKey(key string) string {
	return "__kvs_zh:" + key
}

func (b *Backend) ZHAdd(key, field string, member interface{}, score float64) error {
	_, err := b.Client.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.ZAdd(key, redis.Z{
			Member: field,
			Score:  score,
		}).Err()
		pipe.HSet(zhHashKey(key), field, member).Err()
		return nil
	})
	return err
}

func (b *Backend) ZScore(key string, member interface{}) (*float64, error) {
	if score, err := b.Client.ZScore(key, *keyvaluestore.ToString(member)).Result(); err == nil {
		return &score, nil
	} else if err != redis.Nil {
		return nil, err
	}
	return nil, nil
}

func (b *Backend) ZRem(key string, member interface{}) error {
	return b.Client.ZRem(key, member).Err()
}

func (b *Backend) ZHRem(key, field string) error {
	_, err := b.Client.TxPipelined(func(pipe redis.Pipeliner) error {
		pipe.ZRem(key, field).Err()
		pipe.HDel(zhHashKey(key), field).Err()
		return nil
	})
	return err
}

func (b *Backend) ZRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZHRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZHRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	results, err := b.Client.ZRangeByScoreWithScores(key, redis.ZRangeBy{
		Min:   strings.ToLower(strconv.FormatFloat(min, 'g', -1, 64)),
		Max:   strings.ToLower(strconv.FormatFloat(max, 'g', -1, 64)),
		Count: int64(limit),
	}).Result()

	if err != nil {
		return nil, err
	}

	members := make([]*keyvaluestore.ScoredMember, len(results))

	for i, res := range results {
		members[i] = &keyvaluestore.ScoredMember{
			Score: res.Score,
			Value: res.Member.(string),
		}
	}

	return members, nil
}

func (b *Backend) ZHRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zhRangeByScoreWithScores("zrangebyscore", key, min, max, limit)
}

func (b *Backend) zhRangeByScoreWithScores(cmd, key string, start, end float64, limit int) (keyvaluestore.ScoredMembers, error) {
	args := []interface{}{start, end, "WITHSCORES"}
	if limit != 0 {
		args = append(args, "LIMIT", 0, limit)
	}
	result, err := b.Client.Eval(`
		local m = redis.call('`+cmd+`', KEYS[1], unpack(ARGV))
		if #m == 0 then return {} end
		local f = {}
		for i=1,#m/2 do f[i]=m[i*2-1] end
		local v = redis.call('hmget', KEYS[2], unpack(f))
		for i,v in pairs(v) do if v then m[i*2-1]=v end end
		return m
	`,
		[]string{key, zhHashKey(key)},
		args...,
	).Result()
	if err != nil {
		return nil, err
	}

	results := result.([]interface{})
	members := make([]*keyvaluestore.ScoredMember, len(results)/2)

	for i := range members {
		score, err := strconv.ParseFloat(results[i*2+1].(string), 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing score: %w", err)
		}
		members[i] = &keyvaluestore.ScoredMember{
			Score: score,
			Value: results[i*2].(string),
		}
	}

	return members, nil
}

func (b *Backend) ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZRevRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZHRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZHRevRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	results, err := b.Client.ZRevRangeByScoreWithScores(key, redis.ZRangeBy{
		Min:   strings.ToLower(strconv.FormatFloat(min, 'g', -1, 64)),
		Max:   strings.ToLower(strconv.FormatFloat(max, 'g', -1, 64)),
		Count: int64(limit),
	}).Result()

	if err != nil {
		return nil, err
	}

	members := make([]*keyvaluestore.ScoredMember, len(results))

	for i, res := range results {
		members[i] = &keyvaluestore.ScoredMember{
			Score: res.Score,
			Value: res.Member.(string),
		}
	}

	return members, nil
}

func (b *Backend) ZHRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zhRangeByScoreWithScores("zrevrangebyscore", key, max, min, limit)
}

func (b *Backend) ZCount(key string, min, max float64) (int, error) {
	n, err := b.Client.ZCount(key,
		strings.ToLower(strconv.FormatFloat(min, 'g', -1, 64)),
		strings.ToLower(strconv.FormatFloat(max, 'g', -1, 64)),
	).Result()
	return int(n), err
}

func (b *Backend) ZLexCount(key string, min, max string) (int, error) {
	n, err := b.Client.ZLexCount(key, min, max).Result()
	return int(n), err
}

func (b *Backend) ZRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.Client.ZRangeByLex(key, redis.ZRangeBy{
		Min:   min,
		Max:   max,
		Count: int64(limit),
	}).Result()
}

func (b *Backend) ZHRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.zhRangeByLex("zrangebylex", key, min, max, limit)
}

func (b *Backend) zhRangeByLex(cmd, key string, start, end string, limit int) ([]string, error) {
	args := []interface{}{start, end}
	if limit != 0 {
		args = append(args, "LIMIT", 0, limit)
	}
	result, err := b.Client.Eval(`
		local f = redis.call('`+cmd+`', KEYS[1], unpack(ARGV))
		if #f == 0 then return {} end
		for i,v in pairs(redis.call('hmget', KEYS[2], unpack(f))) do if v then f[i] = v end end
		return f
	`,
		[]string{key, zhHashKey(key)},
		args...,
	).Result()
	if err != nil {
		return nil, err
	}
	values := result.([]interface{})
	ret := make([]string, len(values))
	for i, v := range values {
		ret[i] = v.(string)
	}
	return ret, nil
}

func (b *Backend) ZRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.Client.ZRevRangeByLex(key, redis.ZRangeBy{
		Min:   min,
		Max:   max,
		Count: int64(limit),
	}).Result()
}

func (b *Backend) ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.zhRangeByLex("zrevrangebylex", key, max, min, limit)
}

func (b *Backend) WithProfiler(profiler interface{}) keyvaluestore.Backend {
	if p, ok := profiler.(Profiler); ok {
		return &Backend{
			Client: ProfileClient(b.Client, p),
		}
	}
	return b
}

func (b *Backend) WithEventuallyConsistentReads() keyvaluestore.Backend {
	return b
}

func (b *Backend) Unwrap() keyvaluestore.Backend {
	return nil
}
