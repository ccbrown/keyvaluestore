package foundationdbstore

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/apple/foundationdb/bindings/go/src/fdb/subspace"
	"github.com/apple/foundationdb/bindings/go/src/fdb/tuple"

	"github.com/ccbrown/keyvaluestore"
)

type Database interface {
	Transact(f func(fdb.Transaction) (interface{}, error)) (interface{}, error)
	ReadTransact(f func(fdb.ReadTransaction) (interface{}, error)) (interface{}, error)
}

type Backend struct {
	Database Database
	Subspace subspace.Subspace
}

func (b *Backend) key(key string) fdb.Key {
	return b.Subspace.Pack(tuple.Tuple{key})
}

func (b *Backend) WithProfiler(profiler interface{}) keyvaluestore.Backend {
	if p, ok := profiler.(Profiler); ok {
		ret := *b
		ret.Database = &ProfilingDatabase{
			Database: b.Database,
			Profiler: p,
		}
		return &ret
	}
	return b
}

func (b *Backend) WithEventuallyConsistentReads() keyvaluestore.Backend {
	return b
}

func (b *Backend) AtomicWrite() keyvaluestore.AtomicWriteOperation {
	return &AtomicWriteOperation{
		Backend: b,
	}
}

func (b *Backend) Batch() keyvaluestore.BatchOperation {
	return &BatchOperation{
		FallbackBatchOperation: &keyvaluestore.FallbackBatchOperation{
			Backend: b,
		},
		Backend: b,
	}
}

func toBytes(v interface{}) []byte {
	switch v := v.(type) {
	case []byte:
		return v
	case string:
		return []byte(v)
	case int:
		return toBytes(int64(v))
	case int64:
		return []byte(strconv.FormatInt(v, 10))
	case encoding.BinaryMarshaler:
		b, err := v.MarshalBinary()
		if err != nil {
			panic("binary marshaler values shouldn't panic. error: " + err.Error())
		}
		return b
	}
	panic(fmt.Sprintf("unsupported value type: %T", v))
}

func (b *Backend) NIncrBy(key string, n int64) (int64, error) {
	if r, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return b.nIncrBy(tx, key, n)
	}); err != nil {
		return 0, err
	} else {
		return r.(int64), nil
	}
}

func (b *Backend) nIncrBy(tx fdb.Transaction, key string, n int64) (int64, error) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(n))
	k := b.key(key)
	tx.Add(k, buf[:])
	r, err := tx.Get(b.key(key)).Get()
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(r)), nil
}

func (b *Backend) Delete(key string) (bool, error) {
	if didDelete, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return b.delete(tx, key)
	}); err != nil {
		return false, err
	} else {
		return didDelete.(bool), nil
	}
}

func (b *Backend) delete(tx fdb.Transaction, key string) (bool, error) {
	k := b.key(key)
	v, err := tx.Get(k).Get()
	if err != nil || v == nil {
		return false, err
	}
	tx.Clear(k)
	return true, nil
}

func (b *Backend) Get(key string) (*string, error) {
	if r, err := b.Database.ReadTransact(func(tx fdb.ReadTransaction) (interface{}, error) {
		return tx.Get(b.key(key)).Get()
	}); err != nil {
		return nil, err
	} else if b := r.([]byte); b != nil {
		s := string(b)
		return &s, nil
	}
	return nil, nil
}

func (b *Backend) Set(key string, value interface{}) error {
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		tx.Set(b.key(key), toBytes(value))
		return nil, nil
	})
	return err
}

func (b *Backend) SetNX(key string, value interface{}) (bool, error) {
	if didSet, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return b.setNX(tx, key, value)
	}); err != nil {
		return false, err
	} else {
		return didSet.(bool), nil
	}
}

func (b *Backend) setNX(tx fdb.Transaction, key string, value interface{}) (bool, error) {
	k := b.key(key)
	v, err := tx.Get(k).Get()
	if err != nil || v != nil {
		return false, err
	}
	tx.Set(k, toBytes(value))
	return true, nil
}

func (b *Backend) SetXX(key string, value interface{}) (bool, error) {
	if didSet, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return b.setXX(tx, key, value)
	}); err != nil {
		return false, err
	} else {
		return didSet.(bool), nil
	}
}

func (b *Backend) setXX(tx fdb.Transaction, key string, value interface{}) (bool, error) {
	k := b.key(key)
	v, err := tx.Get(k).Get()
	if err != nil || v == nil {
		return false, err
	}
	tx.Set(k, toBytes(value))
	return true, nil
}

func (b *Backend) SetEQ(key string, value, oldValue interface{}) (bool, error) {
	if didSet, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		return b.setEQ(tx, key, value, oldValue)
	}); err != nil {
		return false, err
	} else {
		return didSet.(bool), nil
	}
}

func (b *Backend) setEQ(tx fdb.Transaction, key string, value, oldValue interface{}) (bool, error) {
	k := b.key(key)
	v, err := tx.Get(k).Get()
	if err != nil || !bytes.Equal(v, toBytes(oldValue)) {
		return false, err
	}
	tx.Set(k, toBytes(value))
	return true, nil
}

func (b *Backend) SAdd(key string, member interface{}, members ...interface{}) error {
	toAdd := make(map[string]struct{}, 1+len(members))
	toAdd[string(toBytes(member))] = struct{}{}
	for _, member := range members {
		toAdd[string(toBytes(member))] = struct{}{}
	}
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		op := sAdd{B: b}
		op.InitNonBlocking(tx, key)
		return nil, op.Complete(tx, key, toAdd)
	})
	return err
}

type sAdd struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *sAdd) InitNonBlocking(tx fdb.Transaction, key string) {
	op.get = tx.Get(op.B.key(key))
}

func (op *sAdd) Complete(tx fdb.Transaction, key string, toAdd map[string]struct{}) error {
	v, err := op.get.Get()
	if err != nil {
		return err
	}
	rem := v
	for len(rem) > 0 {
		l, n := binary.Uvarint(rem)
		if n <= 0 || uint64(len(rem)) < uint64(n)+l {
			return fmt.Errorf("unable to decode set")
		}
		delete(toAdd, string(rem[n:n+int(l)]))
		rem = rem[n+int(l):]
	}
	if len(toAdd) > 0 {
		newValue := append([]byte(nil), v...)
		for member := range toAdd {
			var buf [binary.MaxVarintLen64]byte
			b := []byte(member)
			n := binary.PutUvarint(buf[:], uint64(len(b)))
			newValue = append(newValue, buf[:n]...)
			newValue = append(newValue, b...)
		}
		tx.Set(op.B.key(key), newValue)
	}
	return nil
}

func (b *Backend) SRem(key string, member interface{}, members ...interface{}) error {
	toRem := make(map[string]struct{}, 1+len(members))
	toRem[string(toBytes(member))] = struct{}{}
	for _, member := range members {
		toRem[string(toBytes(member))] = struct{}{}
	}
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		op := sRem{B: b}
		op.InitNonBlocking(tx, key)
		return nil, op.Complete(tx, key, toRem)
	})
	return err
}

type sRem struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *sRem) InitNonBlocking(tx fdb.Transaction, key string) {
	op.get = tx.Get(op.B.key(key))
}

func (op *sRem) Complete(tx fdb.Transaction, key string, toRem map[string]struct{}) error {
	v, err := op.get.Get()
	if err != nil {
		return err
	}
	var newValue []byte
	rem := v
	for len(rem) > 0 {
		l, n := binary.Uvarint(rem)
		if n <= 0 || uint64(len(rem)) < uint64(n)+l {
			return fmt.Errorf("unable to decode set")
		}
		if _, ok := toRem[string(rem[n:n+int(l)])]; !ok {
			newValue = append(newValue, rem[:n+int(l)]...)
		}
		rem = rem[n+int(l):]
	}
	if len(newValue) < len(v) {
		tx.Set(op.B.key(key), newValue)
	}
	return nil
}

func (b *Backend) SMembers(key string) ([]string, error) {
	if r, err := b.Database.ReadTransact(func(tx fdb.ReadTransaction) (interface{}, error) {
		return tx.Get(b.key(key)).Get()
	}); err != nil {
		return nil, err
	} else if b := r.([]byte); b != nil {
		return parseSMembers(b)
	}
	return nil, nil
}

func parseSMembers(b []byte) ([]string, error) {
	var ret []string
	for len(b) > 0 {
		l, n := binary.Uvarint(b)
		if n <= 0 || uint64(len(b)) < uint64(n)+l {
			return nil, fmt.Errorf("unable to decode set")
		}
		ret = append(ret, string(b[n:n+int(l)]))
		b = b[n+int(l):]
	}
	return ret, nil
}

func (b *Backend) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) error {
	toAdd := make(map[string]interface{}, 1+len(fields))
	toAdd[field] = value
	for _, field := range fields {
		toAdd[field.Key] = field.Value
	}
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		impl := hSet{B: b}
		impl.InitNonBlocking(tx, key)
		return nil, impl.Complete(tx, key, toAdd)
	})
	return err
}

type hSet struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *hSet) InitNonBlocking(tx fdb.Transaction, key string) {
	op.get = tx.Get(op.B.key(key))
}

func (op *hSet) Complete(tx fdb.Transaction, key string, toAdd map[string]interface{}) error {
	v, err := op.get.Get()
	if err != nil {
		return err
	}
	var newValue []byte
	rem := v
	for len(rem) > 0 {
		kl, kn := binary.Uvarint(rem)
		if kn <= 0 || uint64(len(rem)) < uint64(kn)+kl {
			return fmt.Errorf("unable to decode hash")
		}
		vl, vn := binary.Uvarint(rem[kn+int(kl):])
		if vn <= 0 || uint64(len(rem)) < uint64(kn+vn)+kl+vl {
			return fmt.Errorf("unable to decode hash")
		}
		if _, ok := toAdd[string(rem[kn:kn+int(kl)])]; !ok {
			newValue = append(newValue, rem[:kn+vn+int(kl+vl)]...)
		}
		rem = rem[kn+vn+int(kl+vl):]
	}
	for key, value := range toAdd {
		var buf [binary.MaxVarintLen64]byte
		kb := []byte(key)
		n := binary.PutUvarint(buf[:], uint64(len(kb)))
		newValue = append(newValue, buf[:n]...)
		newValue = append(newValue, kb...)
		vb := toBytes(value)
		n = binary.PutUvarint(buf[:], uint64(len(vb)))
		newValue = append(newValue, buf[:n]...)
		newValue = append(newValue, vb...)
	}
	tx.Set(op.B.key(key), newValue)
	return nil
}

func (op *hSet) CompleteNX(tx fdb.Transaction, key, field string, value interface{}) (bool, error) {
	v, err := op.get.Get()
	if err != nil {
		return false, err
	}
	rem := v
	for len(rem) > 0 {
		kl, kn := binary.Uvarint(rem)
		if kn <= 0 || uint64(len(rem)) < uint64(kn)+kl {
			return false, fmt.Errorf("unable to decode hash")
		}
		vl, vn := binary.Uvarint(rem[kn+int(kl):])
		if vn <= 0 || uint64(len(rem)) < uint64(kn+vn)+kl+vl {
			return false, fmt.Errorf("unable to decode hash")
		}
		if string(rem[kn:kn+int(kl)]) == field {
			return false, nil
		}
		rem = rem[kn+vn+int(kl+vl):]
	}

	var buf [binary.MaxVarintLen64]byte
	kb := []byte(field)
	n := binary.PutUvarint(buf[:], uint64(len(kb)))
	vb := toBytes(value)
	newValue := make([]byte, len(v)+16+len(kb)+len(vb))
	newValue = append(newValue, v...)
	newValue = append(newValue, buf[:n]...)
	newValue = append(newValue, kb...)
	n = binary.PutUvarint(buf[:], uint64(len(vb)))
	newValue = append(newValue, buf[:n]...)
	newValue = append(newValue, vb...)

	tx.Set(op.B.key(key), newValue)
	return true, nil
}

func (b *Backend) HDel(key, field string, fields ...string) error {
	toDel := make(map[string]struct{}, 1+len(fields))
	toDel[field] = struct{}{}
	for _, field := range fields {
		toDel[field] = struct{}{}
	}
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		impl := &hDel{B: b}
		impl.InitNonBlocking(tx, key)
		return nil, impl.Complete(tx, key, toDel)
	})
	return err
}

type hDel struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *hDel) InitNonBlocking(tx fdb.Transaction, key string) {
	op.get = tx.Get(op.B.key(key))
}

func (op *hDel) Complete(tx fdb.Transaction, key string, toDel map[string]struct{}) error {
	v, err := op.get.Get()
	if err != nil {
		return err
	}
	var newValue []byte
	rem := v
	for len(rem) > 0 {
		kl, kn := binary.Uvarint(rem)
		if kn <= 0 || uint64(len(rem)) < uint64(kn)+kl {
			return fmt.Errorf("unable to decode hash")
		}
		vl, vn := binary.Uvarint(rem[kn+int(kl):])
		if vn <= 0 || uint64(len(rem)) < uint64(kn+vn)+kl+vl {
			return fmt.Errorf("unable to decode hash")
		}
		if _, ok := toDel[string(rem[kn:kn+int(kl)])]; !ok {
			newValue = append(newValue, rem[:kn+vn+int(kl+vl)]...)
		}
		rem = rem[kn+vn+int(kl+vl):]
	}
	if len(newValue) < len(v) {
		tx.Set(op.B.key(key), newValue)
	}
	return nil
}

func (b *Backend) HGet(key, field string) (*string, error) {
	if all, err := b.HGetAll(key); err != nil {
		return nil, err
	} else if v, ok := all[field]; ok {
		return &v, nil
	}
	return nil, nil
}

func (b *Backend) HGetAll(key string) (map[string]string, error) {
	k := b.key(key)
	if r, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		b, err := tx.Get(k).Get()
		if err != nil {
			return nil, err
		}
		rem := b
		ret := map[string]string{}
		for len(rem) > 0 {
			kl, kn := binary.Uvarint(rem)
			if kn <= 0 || uint64(len(rem)) < uint64(kn)+kl {
				return nil, fmt.Errorf("unable to decode hash")
			}
			vl, vn := binary.Uvarint(rem[kn+int(kl):])
			if vn <= 0 || uint64(len(rem)) < uint64(kn+vn)+kl+vl {
				return nil, fmt.Errorf("unable to decode hash")
			}
			ret[string(rem[kn:kn+int(kl)])] = string(rem[kn+int(kl)+vn : kn+vn+int(kl+vl)])
			rem = rem[kn+vn+int(kl+vl):]
		}
		return ret, nil
	}); err != nil {
		return nil, err
	} else {
		return r.(map[string]string), nil
	}
}

func (b *Backend) ZAdd(key string, member interface{}, score float64) error {
	s := *keyvaluestore.ToString(member)
	return b.ZHAdd(key, s, s, score)
}

func (b *Backend) zLexKey(key, field string) fdb.Key {
	return b.Subspace.Pack(tuple.Tuple{key, "l", field})
}

func (b *Backend) zScoreKey(key, field string, score float64) fdb.Key {
	return b.Subspace.Pack(tuple.Tuple{key, "s", score, field})
}

func floatBytes(f float64) []byte {
	n := math.Float64bits(f)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	return buf
}

func floatFromBytes(b []byte) float64 {
	if len(b) < 8 {
		return 0
	}
	n := binary.BigEndian.Uint64(b)
	return math.Float64frombits(n)
}

func (b *Backend) ZHAdd(key, field string, member interface{}, score float64) error {
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		op := zHAdd{B: b}
		op.InitNonBlocking(tx, key, field)
		return nil, op.Complete(tx, key, field, member, score)
	})
	return err
}

type zHAdd struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *zHAdd) InitNonBlocking(tx fdb.Transaction, key, field string) {
	op.get = tx.Get(op.B.zLexKey(key, field))
}

func (op *zHAdd) Complete(tx fdb.Transaction, key, field string, member interface{}, score float64) error {
	v := toBytes(member)
	existing, err := op.get.Get()
	if err != nil {
		return err
	}
	if existing != nil {
		if prevScore := floatFromBytes(existing[:8]); prevScore != score {
			tx.Clear(op.B.zScoreKey(key, field, prevScore))
		}
	}
	tx.Set(op.B.zLexKey(key, field), append(floatBytes(score), v...))
	tx.Set(op.B.zScoreKey(key, field, score), v)
	return err
}

func (op *zHAdd) CompleteNX(tx fdb.Transaction, key, field string, member interface{}, score float64) (bool, error) {
	v := toBytes(member)
	existing, err := op.get.Get()
	if err != nil || existing != nil {
		return false, err
	}
	tx.Set(op.B.zLexKey(key, field), append(floatBytes(score), v...))
	tx.Set(op.B.zScoreKey(key, field, score), v)
	return true, err
}

func (b *Backend) zHAddNX(tx fdb.Transaction, key, field string, member interface{}, score float64) (bool, error) {
	if r, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		op := zHAdd{B: b}
		op.InitNonBlocking(tx, key, field)
		return op.CompleteNX(tx, key, field, member, score)
	}); err != nil {
		return false, err
	} else {
		return r.(bool), err
	}
}

func (b *Backend) ZScore(key string, member interface{}) (*float64, error) {
	if r, err := b.Database.ReadTransact(func(tx fdb.ReadTransaction) (interface{}, error) {
		return b.zScore(tx, key, member)
	}); err != nil {
		return nil, err
	} else {
		return r.(*float64), nil
	}
}

func (b *Backend) zScore(tx fdb.ReadTransaction, key string, member interface{}) (*float64, error) {
	field := *keyvaluestore.ToString(member)
	existing, err := tx.Get(b.zLexKey(key, field)).Get()
	if err != nil || len(existing) < 8 {
		return nil, err
	}
	score := floatFromBytes(existing[:8])
	return &score, nil
}

func (b *Backend) ZIncrBy(key string, member string, n float64) (float64, error) {
	field := *keyvaluestore.ToString(member)
	v := []byte(field)
	k := b.zLexKey(key, field)
	if score, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		score := n
		if existing, err := tx.Get(k).Get(); err != nil {
			return nil, err
		} else if existing != nil {
			prevScore := floatFromBytes(existing[:8])
			score += prevScore
			tx.Clear(b.zScoreKey(key, field, prevScore))
		}
		tx.Set(k, append(floatBytes(score), v...))
		tx.Set(b.zScoreKey(key, field, score), v)
		return score, nil
	}); err != nil {
		return 0.0, err
	} else {
		return score.(float64), nil
	}
}

func (b *Backend) ZRem(key string, member interface{}) error {
	s := *keyvaluestore.ToString(member)
	return b.ZHRem(key, s)
}

func (b *Backend) ZHRem(key, field string) error {
	_, err := b.Database.Transact(func(tx fdb.Transaction) (interface{}, error) {
		op := zHRem{B: b}
		op.InitNonBlocking(tx, key, field)
		return nil, op.Complete(tx, key, field)
	})
	return err
}

type zHRem struct {
	B   *Backend
	get fdb.FutureByteSlice
}

func (op *zHRem) InitNonBlocking(tx fdb.Transaction, key, field string) {
	k := op.B.zLexKey(key, field)
	op.get = tx.Get(k)
	tx.Clear(k)
}

func (op *zHRem) Complete(tx fdb.Transaction, key, field string) error {
	existing, err := op.get.Get()
	if err == nil && existing != nil {
		score := floatFromBytes(existing[:8])
		tx.Clear(op.B.zScoreKey(key, field, score))
	}
	return err
}

func (b *Backend) ZCount(key string, min, max float64) (int, error) {
	// TODO: use the (also iffy) approach here?:
	// https://forums.foundationdb.org/t/getting-the-number-of-key-value-pairs/189/5
	s, err := b.ZRangeByScore(key, min, max, 0)
	return len(s), err
}

func (b *Backend) ZLexCount(key, min, max string) (int, error) {
	// TODO: use the (also iffy) approach here?:
	// https://forums.foundationdb.org/t/getting-the-number-of-key-value-pairs/189/5
	s, err := b.ZRangeByLex(key, min, max, 0)
	return len(s), err
}

func (b *Backend) ZRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZHRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return b.ZRangeByScore(key, min, max, limit)
}

func (b *Backend) ZRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zRangeByScoreWithScores(key, min, max, limit, false)
}

func (b *Backend) scoreRange(key string, min, max float64) fdb.Range {
	var begin fdb.KeySelector
	if min == math.Inf(-1) {
		begin = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "s"}))
	} else {
		begin = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "s", min}))
	}

	var end fdb.KeySelector
	if max == math.Inf(1) {
		end = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "t"}))
	} else {
		end = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "s", math.Nextafter(max, math.Inf(1))}))
	}

	return fdb.SelectorRange{
		Begin: begin,
		End:   end,
	}
}

func (b *Backend) zRangeByScoreWithScores(key string, min, max float64, limit int, reverse bool) (keyvaluestore.ScoredMembers, error) {
	if r, err := b.Database.ReadTransact(func(tx fdb.ReadTransaction) (interface{}, error) {
		it := tx.GetRange(
			b.scoreRange(key, min, max),
			fdb.RangeOptions{
				Mode:    fdb.StreamingModeWantAll,
				Limit:   limit,
				Reverse: reverse,
			},
		).Iterator()
		var ret keyvaluestore.ScoredMembers
		for it.Advance() {
			kv, err := it.Get()
			if err != nil {
				return nil, err
			}
			key, err := b.Subspace.Unpack(kv.Key)
			if err != nil {
				return nil, err
			}
			ret = append(ret, &keyvaluestore.ScoredMember{
				Score: key[2].(float64),
				Value: string(kv.Value),
			})
		}
		return ret, nil
	}); err != nil {
		return nil, err
	} else {
		return r.(keyvaluestore.ScoredMembers), nil
	}
}

func (b *Backend) ZHRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.ZRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	members, err := b.ZRevRangeByScoreWithScores(key, min, max, limit)
	return members.Values(), err
}

func (b *Backend) ZHRevRangeByScore(key string, min, max float64, limit int) ([]string, error) {
	return b.ZRevRangeByScore(key, min, max, limit)
}

func (b *Backend) ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.zRangeByScoreWithScores(key, min, max, limit, true)
}

func (b *Backend) ZHRevRangeByScoreWithScores(key string, min, max float64, limit int) (keyvaluestore.ScoredMembers, error) {
	return b.ZRevRangeByScoreWithScores(key, min, max, limit)
}

func (b *Backend) ZRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.ZHRangeByLex(key, min, max, limit)
}

func (b *Backend) ZHRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.zHRangeByLex(key, min, max, limit, false)
}

func (b *Backend) lexRange(key string, min, max string) fdb.Range {
	var begin fdb.KeySelector
	if min[0] == '-' {
		begin = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "l"}))
	} else if min[0] == '[' {
		begin = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "l", min[1:]}))
	} else {
		begin = fdb.FirstGreaterThan(b.Subspace.Pack(tuple.Tuple{key, "l", min[1:]}))
	}

	var end fdb.KeySelector
	if max[0] == '+' {
		end = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "m"}))
	} else if max[0] == '[' {
		end = fdb.FirstGreaterThan(b.Subspace.Pack(tuple.Tuple{key, "l", max[1:]}))
	} else {
		end = fdb.FirstGreaterOrEqual(b.Subspace.Pack(tuple.Tuple{key, "l", max[1:]}))
	}

	return fdb.SelectorRange{
		Begin: begin,
		End:   end,
	}
}

func (b *Backend) zHRangeByLex(key string, min, max string, limit int, reverse bool) ([]string, error) {
	if r, err := b.Database.ReadTransact(func(tx fdb.ReadTransaction) (interface{}, error) {
		it := tx.GetRange(
			b.lexRange(key, min, max),
			fdb.RangeOptions{
				Mode:    fdb.StreamingModeWantAll,
				Limit:   limit,
				Reverse: reverse,
			},
		).Iterator()
		var ret []string
		for it.Advance() {
			kv, err := it.Get()
			if err != nil {
				return nil, err
			}
			ret = append(ret, string(kv.Value[8:]))
		}
		return ret, nil
	}); err != nil {
		return nil, err
	} else {
		return r.([]string), nil
	}
}

func (b *Backend) ZRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.ZHRevRangeByLex(key, min, max, limit)
}

func (b *Backend) ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error) {
	return b.zHRangeByLex(key, min, max, limit, true)
}

func (b *Backend) Unwrap() keyvaluestore.Backend {
	return nil
}
