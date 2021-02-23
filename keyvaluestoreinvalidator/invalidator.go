package keyvaluestoreinvalidator

import (
	"github.com/ccbrown/keyvaluestore"
)

// Invalidate passes operations through to an underlying backend and invokes a specified function
// for keys that may have been impacted as a result.
type Invalidator struct {
	Backend    keyvaluestore.Backend
	Invalidate func(key string)
}

var _ keyvaluestore.Backend = &Invalidator{}

func (c *Invalidator) AtomicWrite() keyvaluestore.AtomicWriteOperation {
	return &atomicWriteOperation{
		invalidator: c,
		atomicWrite: c.Backend.AtomicWrite(),
	}
}

func (c *Invalidator) Batch() keyvaluestore.BatchOperation {
	return &batchOperation{
		invalidator: c,
		batch:       c.Backend.Batch(),
	}
}

func (c *Invalidator) Delete(key string) (success bool, err error) {
	success, err = c.Backend.Delete(key)
	c.Invalidate(key)
	return success, err
}

func (c *Invalidator) Get(key string) (*string, error) {
	return c.Backend.Get(key)
}

func (c *Invalidator) Set(key string, value interface{}) error {
	err := c.Backend.Set(key, value)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) NIncrBy(key string, n int64) (int64, error) {
	n, err := c.Backend.NIncrBy(key, n)
	c.Invalidate(key)
	return n, err
}

func (c *Invalidator) SetXX(key string, value interface{}) (bool, error) {
	ok, err := c.Backend.SetXX(key, value)
	c.Invalidate(key)
	return ok, err
}

func (c *Invalidator) SetNX(key string, value interface{}) (bool, error) {
	ok, err := c.Backend.SetNX(key, value)
	c.Invalidate(key)
	return ok, err
}

func (c *Invalidator) SetEQ(key string, value, oldValue interface{}) (bool, error) {
	ok, err := c.Backend.SetEQ(key, value, oldValue)
	c.Invalidate(key)
	return ok, err
}

func (c *Invalidator) SAdd(key string, member interface{}, members ...interface{}) error {
	err := c.Backend.SAdd(key, member, members...)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) SRem(key string, member interface{}, members ...interface{}) error {
	err := c.Backend.SRem(key, member, members...)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	err := c.Backend.HSet(key, field, value, fields...)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) HDel(key, field string, fields ...string) error {
	err := c.Backend.HDel(key, field, fields...)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) HGet(key, field string) (*string, error) {
	return c.Backend.HGet(key, field)
}

func (c *Invalidator) HGetAll(key string) (map[string]string, error) {
	return c.Backend.HGetAll(key)
}

func (c *Invalidator) SMembers(key string) ([]string, error) {
	return c.Backend.SMembers(key)
}

func (c *Invalidator) ZAdd(key string, member interface{}, score float64) error {
	err := c.Backend.ZAdd(key, member, score)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) ZHAdd(key, field string, member interface{}, score float64) error {
	err := c.Backend.ZHAdd(key, field, member, score)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) ZScore(key string, member interface{}) (*float64, error) {
	return c.Backend.ZScore(key, member)
}

func (c *Invalidator) ZIncrBy(key string, member string, n float64) (float64, error) {
	val, err := c.Backend.ZIncrBy(key, member, n)
	c.Invalidate(key)
	return val, err
}

func (c *Invalidator) ZRem(key string, member interface{}) error {
	err := c.Backend.ZRem(key, member)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) ZHRem(key, field string) error {
	err := c.Backend.ZHRem(key, field)
	c.Invalidate(key)
	return err
}

func (c *Invalidator) ZCount(key string, min, max float64) (int, error) {
	return c.Backend.ZCount(key, min, max)
}

func (c *Invalidator) ZLexCount(key string, min, max string) (int, error) {
	return c.Backend.ZLexCount(key, min, max)
}

func (c *Invalidator) ZRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return c.Backend.ZRangeByScore(key, min, max, limit)
}

func (c *Invalidator) ZHRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return c.Backend.ZHRangeByScore(key, min, max, limit)
}

func (c *Invalidator) ZRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return c.Backend.ZRangeByScoreWithScores(key, min, max, limit)
}

func (c *Invalidator) ZHRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return c.Backend.ZHRangeByScoreWithScores(key, min, max, limit)
}

func (c *Invalidator) ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return c.Backend.ZRevRangeByScore(key, min, max, limit)
}

func (c *Invalidator) ZHRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return c.Backend.ZHRevRangeByScore(key, min, max, limit)
}

func (c *Invalidator) ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return c.Backend.ZRevRangeByScoreWithScores(key, min, max, limit)
}

func (c *Invalidator) ZHRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return c.Backend.ZHRevRangeByScoreWithScores(key, min, max, limit)
}

func (c *Invalidator) ZRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return c.Backend.ZRangeByLex(key, min, max, limit)
}

func (c *Invalidator) ZHRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return c.Backend.ZHRangeByLex(key, min, max, limit)
}

func (c *Invalidator) ZRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return c.Backend.ZRevRangeByLex(key, min, max, limit)
}

func (c *Invalidator) ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return c.Backend.ZHRevRangeByLex(key, min, max, limit)
}

func (c Invalidator) WithProfiler(profiler interface{}) keyvaluestore.Backend {
	c.Backend = c.Backend.WithProfiler(profiler)
	return &c
}

func (c Invalidator) WithEventuallyConsistentReads() keyvaluestore.Backend {
	c.Backend = c.Backend.WithEventuallyConsistentReads()
	return &c
}

func (c *Invalidator) Unwrap() keyvaluestore.Backend {
	return c.Backend
}
