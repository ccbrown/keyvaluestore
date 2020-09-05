package keyvaluestore

type AtomicWriteResult interface {
	// Returns true if the transaction failed due to this operation's conditional failing.
	ConditionalFailed() bool
}

// DynamoDB can't do more than 25 operations in an atomic write so all backends should enforce this
// limit.
const MaxAtomicWriteOperations = 25

type AtomicWriteOperation interface {
	// Sets a key. No conditionals are applied.
	Set(key string, value interface{}) AtomicWriteResult

	// Sets a key. The atomic write operation will be aborted if the key already exists.
	SetNX(key string, value interface{}) AtomicWriteResult

	// Sets a key. The atomic write operation will be aborted if the key does not already exist.
	SetXX(key string, value interface{}) AtomicWriteResult

	// Sets a key. The atomic write operation will be aborted if the key does not exist or does not
	// have the given value.
	SetEQ(key string, value, oldValue interface{}) AtomicWriteResult

	// Deletes a key. No conditionals are applied.
	Delete(key string) AtomicWriteResult

	// Deletes a key. The atomic write operation will be aborted if the key does not exist.
	DeleteXX(key string) AtomicWriteResult

	// Increments the given key by some number. If the key doesn't exist, it's set to the given
	// number instead. No conditionals are applied.
	IncrBy(key string, n int64) AtomicWriteResult

	// Adds a member to a sorted set. No conditionals are applied.
	ZAdd(key string, member interface{}, score float64) AtomicWriteResult

	// Removes a member from a sorted set. No conditionals are applied.
	ZRem(key string, member interface{}) AtomicWriteResult

	// Adds a member to a set. No conditionals are applied.
	SAdd(key string, member interface{}, members ...interface{}) AtomicWriteResult

	// Removes a member from a set. No conditionals are applied.
	SRem(key string, member interface{}, members ...interface{}) AtomicWriteResult

	// Executes the operation. If a condition failed, returns false.
	Exec() (bool, error)
}
