package foundationdbstore

import (
	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/ccbrown/keyvaluestore"
)

type BatchOperation struct {
	// We use the fallback for operations that can't be done with snapshot reads.
	*keyvaluestore.FallbackBatchOperation

	Backend *Backend

	// phase one: initiate reads and start non-blocking operations
	p1 []func(tx fdb.Transaction) error

	// phase two: wait for the reads and complete the operations
	p2 []func(tx fdb.Transaction) error
}

type getResult struct {
	v   []byte
	err error
}

func (r *getResult) Result() (*string, error) {
	var v *string
	if r.v != nil {
		s := string(r.v)
		v = &s
	}
	return v, r.err
}

func (op *BatchOperation) Get(key string) keyvaluestore.GetResult {
	r := &getResult{}
	var get fdb.FutureByteSlice
	op.p1 = append(op.p1, func(tx fdb.Transaction) error {
		get = tx.Snapshot().Get(op.Backend.key(key))
		return nil
	})
	op.p2 = append(op.p2, func(tx fdb.Transaction) error {
		r.v, r.err = get.Get()
		return r.err
	})
	return r
}

type sMembersResult struct {
	members []string
	err     error
}

func (r *sMembersResult) Result() ([]string, error) {
	return r.members, r.err
}

func (op *BatchOperation) SMembers(key string) keyvaluestore.SMembersResult {
	r := &sMembersResult{}
	var get fdb.FutureByteSlice
	op.p1 = append(op.p1, func(tx fdb.Transaction) error {
		get = tx.Snapshot().Get(op.Backend.key(key))
		return nil
	})
	op.p2 = append(op.p2, func(tx fdb.Transaction) error {
		var b []byte
		b, r.err = get.Get()
		if r.err == nil {
			r.members, r.err = parseSMembers(b)
		}
		return r.err
	})
	return r
}

type zScoreResult struct {
	score *float64
	err   error
}

func (r *zScoreResult) Result() (*float64, error) {
	return r.score, r.err
}

func (op *BatchOperation) ZScore(key string, member interface{}) keyvaluestore.ZScoreResult {
	field := *keyvaluestore.ToString(member)
	r := &zScoreResult{}
	var get fdb.FutureByteSlice
	op.p1 = append(op.p1, func(tx fdb.Transaction) error {
		get = tx.Snapshot().Get(op.Backend.zLexKey(key, field))
		return nil
	})
	op.p2 = append(op.p2, func(tx fdb.Transaction) error {
		var existing []byte
		existing, r.err = get.Get()
		if r.err != nil || len(existing) < 8 {
			return r.err
		}
		score := floatFromBytes(existing[:8])
		r.score = &score
		return nil
	})
	return r
}

func (op *BatchOperation) Exec() error {
	if _, err := op.Backend.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		for _, f := range op.p1 {
			if err := f(tx); err != nil {
				return nil, err
			}
		}
		for _, f := range op.p2 {
			if err := f(tx); err != nil {
				return nil, err
			}
		}
		return true, nil
	}); err != nil {
		return err
	}
	return op.FallbackBatchOperation.Exec()
}
