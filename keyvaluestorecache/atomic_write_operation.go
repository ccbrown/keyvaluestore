package keyvaluestorecache

import "github.com/ccbrown/keyvaluestore"

type readCacheAtomicWriteOperation struct {
	ReadCache   *ReadCache
	atomicWrite keyvaluestore.AtomicWriteOperation

	invalidations []string
}

func (op *readCacheAtomicWriteOperation) Set(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.Set(key, value)
}

func (op *readCacheAtomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetNX(key, value)
}

func (op *readCacheAtomicWriteOperation) SetXX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetXX(key, value)
}

func (op *readCacheAtomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetEQ(key, value, oldValue)
}

func (op *readCacheAtomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.Delete(key)
}

func (op *readCacheAtomicWriteOperation) DeleteXX(key string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.DeleteXX(key)
}

func (op *readCacheAtomicWriteOperation) IncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.IncrBy(key, n)
}

func (op *readCacheAtomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZAdd(key, member, score)
}

func (op *readCacheAtomicWriteOperation) ZHAdd(key, field string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZHAdd(key, field, member, score)
}

func (op *readCacheAtomicWriteOperation) ZAddNX(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZAddNX(key, member, score)
}

func (op *readCacheAtomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZRem(key, member)
}

func (op *readCacheAtomicWriteOperation) ZHRem(key, field string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZHRem(key, field)
}

func (op *readCacheAtomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SAdd(key, member, members...)
}

func (op *readCacheAtomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SRem(key, member, members...)
}

func (op *readCacheAtomicWriteOperation) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HSet(key, field, value, fields...)
}

func (op *readCacheAtomicWriteOperation) HSetNX(key, field string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HSetNX(key, field, value)
}

func (op *readCacheAtomicWriteOperation) HDel(key, field string, fields ...string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HDel(key, field, fields...)
}

func (op *readCacheAtomicWriteOperation) Exec() (bool, error) {
	ret, err := op.atomicWrite.Exec()
	for _, key := range op.invalidations {
		op.ReadCache.cache.Delete(key)
	}
	return ret, err
}
