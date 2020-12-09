package keyvaluestoreinvalidator

import "github.com/ccbrown/keyvaluestore"

type batchOperation struct {
	invalidator   *Invalidator
	batch         keyvaluestore.BatchOperation
	invalidations []string
}

func (op *batchOperation) Get(key string) keyvaluestore.GetResult {
	return op.batch.Get(key)
}

func (op *batchOperation) Delete(key string) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.Delete(key)
}

func (op *batchOperation) Set(key string, value interface{}) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.Set(key, value)
}

func (op *batchOperation) SMembers(key string) keyvaluestore.SMembersResult {
	return op.batch.SMembers(key)
}

func (op *batchOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.SAdd(key, member, members...)
}

func (op *batchOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.SRem(key, member, members...)
}

func (op *batchOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.ZAdd(key, member, score)
}

func (op *batchOperation) ZRem(key string, member interface{}) keyvaluestore.ErrorResult {
	op.invalidations = append(op.invalidations, key)
	return op.batch.ZRem(key, member)
}

func (op *batchOperation) ZScore(key string, member interface{}) keyvaluestore.ZScoreResult {
	return op.batch.ZScore(key, member)
}

func (op *batchOperation) Exec() error {
	err := op.batch.Exec()
	for _, key := range op.invalidations {
		op.invalidator.Invalidate(key)
	}
	return err
}
