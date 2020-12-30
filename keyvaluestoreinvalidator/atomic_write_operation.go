package keyvaluestoreinvalidator

import "github.com/ccbrown/keyvaluestore"

type atomicWriteOperation struct {
	invalidator   *Invalidator
	atomicWrite   keyvaluestore.AtomicWriteOperation
	invalidations []string
}

func (op *atomicWriteOperation) Set(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.Set(key, value)
}

func (op *atomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetNX(key, value)
}

func (op *atomicWriteOperation) SetXX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetXX(key, value)
}

func (op *atomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SetEQ(key, value, oldValue)
}

func (op *atomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.Delete(key)
}

func (op *atomicWriteOperation) DeleteXX(key string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.DeleteXX(key)
}

func (op *atomicWriteOperation) IncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.IncrBy(key, n)
}

func (op *atomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZAdd(key, member, score)
}

func (op *atomicWriteOperation) ZHAdd(key, field string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZHAdd(key, field, member, score)
}

func (op *atomicWriteOperation) ZAddNX(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZAddNX(key, member, score)
}

func (op *atomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZRem(key, member)
}

func (op *atomicWriteOperation) ZHRem(key, field string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.ZHRem(key, field)
}

func (op *atomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SAdd(key, member, members...)
}

func (op *atomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.SRem(key, member, members...)
}

func (op *atomicWriteOperation) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HSet(key, field, value, fields...)
}

func (op *atomicWriteOperation) HSetNX(key, field string, value interface{}) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HSetNX(key, field, value)
}

func (op *atomicWriteOperation) HDel(key, field string, fields ...string) keyvaluestore.AtomicWriteResult {
	op.invalidations = append(op.invalidations, key)
	return op.atomicWrite.HDel(key, field, fields...)
}

func (op *atomicWriteOperation) Exec() (bool, error) {
	ret, err := op.atomicWrite.Exec()
	// invalidate everything, always. if the transaction wasn't committed, one of the values
	// probably wasn't what the client was expecting and they may want to refetch it and try again
	for _, key := range op.invalidations {
		op.invalidator.Invalidate(key)
	}
	return ret, err
}
