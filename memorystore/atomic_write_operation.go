package memorystore

import (
	"fmt"

	"github.com/ccbrown/keyvaluestore"
)

type AtomicWriteOperation struct {
	Backend *Backend

	operations []*atomicWriteOperation
}

type atomicWriteOperation struct {
	condition func() bool
	write     func()

	conditionPassed bool
}

func (op *atomicWriteOperation) ConditionalFailed() bool {
	return !op.conditionPassed
}

func (op *AtomicWriteOperation) write(wOp *atomicWriteOperation) keyvaluestore.AtomicWriteResult {
	op.operations = append(op.operations, wOp)
	return wOp
}

func (op *AtomicWriteOperation) Set(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.set(key, value)
		},
	})
}

func (op *AtomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		condition: func() bool {
			return op.Backend.get(key) == nil
		},
		write: func() {
			op.Backend.set(key, value)
		},
	})
}

func (op *AtomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		condition: func() bool {
			v := op.Backend.get(key)
			return v != nil && *v == *keyvaluestore.ToString(oldValue)
		},
		write: func() {
			op.Backend.set(key, value)
		},
	})
}

func (op *AtomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.delete(key)
		},
	})
}

func (op *AtomicWriteOperation) IncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.incrBy(key, n)
		},
	})
}

func (op *AtomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.zadd(key, member, func(previousScore *float64) (float64, error) {
				return score, nil
			})
		},
	})
}

func (op *AtomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.zrem(key, member)
		},
	})
}

func (op *AtomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.sadd(key, member, members...)
		},
	})
}

func (op *AtomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		write: func() {
			op.Backend.srem(key, member, members...)
		},
	})
}

func (op *AtomicWriteOperation) Exec() (bool, error) {
	if len(op.operations) > keyvaluestore.MaxAtomicWriteOperations {
		return false, fmt.Errorf("max operation count exceeded")
	}

	op.Backend.mutex.Lock()
	defer op.Backend.mutex.Unlock()

	allPassed := true

	for _, wOp := range op.operations {
		if wOp.condition == nil {
			wOp.conditionPassed = true
		} else {
			pass := wOp.condition()
			wOp.conditionPassed = pass
			if !pass {
				allPassed = false
			}
		}
	}

	if !allPassed {
		return false, nil
	}

	for _, wOp := range op.operations {
		wOp.write()
	}

	return true, nil
}
