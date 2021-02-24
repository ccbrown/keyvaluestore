package foundationdbstore

import (
	"bytes"
	"encoding/binary"

	"github.com/apple/foundationdb/bindings/go/src/fdb"

	"github.com/ccbrown/keyvaluestore"
)

type AtomicWriteOperation struct {
	Backend *Backend

	ops []*atomicWriteOp
}

type atomicWriteOp struct {
	// phase one: initiate reads and start non-blocking operations
	p1 func(tx fdb.Transaction) error

	// phase two: wait for the reads and complete the operations
	p2 func(tx fdb.Transaction) (ok bool, err error)

	conditionalFailed bool
}

func (op *atomicWriteOp) ConditionalFailed() bool {
	return op.conditionalFailed
}

func (op *AtomicWriteOperation) Set(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			tx.Set(op.Backend.key(key), toBytes(value))
			return nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	k := op.Backend.key(key)
	var get fdb.FutureByteSlice
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			get = tx.Get(k)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			v, err := get.Get()
			if err != nil || v != nil {
				return false, err
			}
			tx.Set(k, toBytes(value))
			return true, nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) SetXX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	k := op.Backend.key(key)
	var get fdb.FutureByteSlice
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			get = tx.Get(k)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			v, err := get.Get()
			if err != nil || v == nil {
				return false, err
			}
			tx.Set(k, toBytes(value))
			return true, nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	k := op.Backend.key(key)
	var get fdb.FutureByteSlice
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			get = tx.Get(k)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			v, err := get.Get()
			if err != nil || !bytes.Equal(v, toBytes(oldValue)) {
				return false, err
			}
			tx.Set(k, toBytes(value))
			return true, nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			tx.Clear(op.Backend.key(key))
			return nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) DeleteXX(key string) keyvaluestore.AtomicWriteResult {
	k := op.Backend.key(key)
	var get fdb.FutureByteSlice
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			get = tx.Get(k)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			if existing, err := get.Get(); existing == nil || err != nil {
				return false, err
			}
			tx.Clear(k)
			return true, nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) NIncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			var buf [8]byte
			binary.LittleEndian.PutUint64(buf[:], uint64(n))
			tx.Add(op.Backend.key(key), buf[:])
			return nil
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	s := *keyvaluestore.ToString(member)
	return op.ZHAdd(key, s, s, score)
}

func (op *AtomicWriteOperation) ZHAdd(key, field string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	impl := zHAdd{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key, field)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, field, member, score)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) ZAddNX(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	field := *keyvaluestore.ToString(member)
	impl := zHAdd{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key, field)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return impl.CompleteNX(tx, key, field, member, score)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	s := *keyvaluestore.ToString(member)
	return op.ZHRem(key, s)
}

func (op *AtomicWriteOperation) ZHRem(key, field string) keyvaluestore.AtomicWriteResult {
	impl := zHRem{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key, field)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, field)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	toAdd := make(map[string]struct{}, 1+len(members))
	toAdd[string(toBytes(member))] = struct{}{}
	for _, member := range members {
		toAdd[string(toBytes(member))] = struct{}{}
	}
	impl := sAdd{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, toAdd)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	toRem := make(map[string]struct{}, 1+len(members))
	toRem[string(toBytes(member))] = struct{}{}
	for _, member := range members {
		toRem[string(toBytes(member))] = struct{}{}
	}
	impl := sRem{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, toRem)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) keyvaluestore.AtomicWriteResult {
	toAdd := make(map[string]interface{}, 1+len(fields))
	toAdd[field] = value
	for _, field := range fields {
		toAdd[field.Key] = field.Value
	}
	impl := hSet{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, toAdd)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) HSetNX(key, field string, value interface{}) keyvaluestore.AtomicWriteResult {
	impl := hSet{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return impl.CompleteNX(tx, key, field, value)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) HDel(key, field string, fields ...string) keyvaluestore.AtomicWriteResult {
	toDel := make(map[string]struct{}, 1+len(fields))
	toDel[field] = struct{}{}
	for _, field := range fields {
		toDel[field] = struct{}{}
	}
	impl := hDel{B: op.Backend}
	subOp := &atomicWriteOp{
		p1: func(tx fdb.Transaction) error {
			impl.InitNonBlocking(tx, key)
			return nil
		},
		p2: func(tx fdb.Transaction) (bool, error) {
			return true, impl.Complete(tx, key, toDel)
		},
	}
	op.ops = append(op.ops, subOp)
	return subOp
}

func (op *AtomicWriteOperation) Exec() (bool, error) {
	if r, err := op.Backend.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		for _, op := range op.ops {
			if err := op.p1(tx); err != nil {
				return nil, err
			}
		}
		for _, op := range op.ops {
			if op.p2 != nil {
				if ok, err := op.p2(tx); err != nil {
					return nil, err
				} else if !ok {
					op.conditionalFailed = true
					tx.Cancel()
					return false, nil
				}
			}
		}
		return true, nil
	}); err != nil {
		if err, ok := err.(fdb.Error); ok {
			switch err.Code {
			case 1010: // not_committed, Transaction not committed due to conflict with another transaction
				return false, &keyvaluestore.AtomicWriteConflictError{
					Err: err,
				}
			case 1025: // transaction_cancelled, Operation aborted because the transaction was cancelled
				return false, nil
			}
		}
		return false, err
	} else {
		return r.(bool), nil
	}
}
