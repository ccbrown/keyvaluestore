package foundationdbstore

import (
	"sync/atomic"
	"time"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
)

type Profiler interface {
	AddFoundationDBTransactionProfile(duration time.Duration)
}

type BasicProfiler struct {
	transactionCount       int64
	transactionNanoseconds int64
}

func (p *BasicProfiler) AddFoundationDBTransactionProfile(duration time.Duration) {
	atomic.AddInt64(&p.transactionCount, 1)
	atomic.AddInt64(&p.transactionNanoseconds, int64(duration/time.Nanosecond))
}

func (p *BasicProfiler) FoundationDBTransactionCount() int {
	return int(atomic.LoadInt64(&p.transactionCount))
}

func (p *BasicProfiler) FoundationDBTransactionDuration() time.Duration {
	return time.Duration(atomic.LoadInt64(&p.transactionNanoseconds)) * time.Nanosecond
}

type ProfilingDatabase struct {
	Database Database
	Profiler Profiler
}

func (db *ProfilingDatabase) Transact(f func(fdb.Transaction) (interface{}, error)) (interface{}, error) {
	startTime := time.Now()
	v, err := db.Database.Transact(f)
	db.Profiler.AddFoundationDBTransactionProfile(time.Since(startTime))
	return v, err
}

func (db *ProfilingDatabase) ReadTransact(f func(fdb.ReadTransaction) (interface{}, error)) (interface{}, error) {
	startTime := time.Now()
	v, err := db.Database.ReadTransact(f)
	db.Profiler.AddFoundationDBTransactionProfile(time.Since(startTime))
	return v, err
}
