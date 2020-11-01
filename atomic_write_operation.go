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

	// Add to or create a sorted set. The size of the member may be limited by some backends (for
	// example, DynamoDB limits it to approximately 1024 bytes). No conditionals are applied.
	ZAdd(key string, member interface{}, score float64) AtomicWriteResult

	// Adds a member to a sorted set. The atomic write operation will be aborted if the member
	// already exists in the set.
	ZAddNX(key string, member interface{}, score float64) AtomicWriteResult

	// Removes a member from a sorted set. No conditionals are applied.
	ZRem(key string, member interface{}) AtomicWriteResult

	// Add to or create a sorted hash. A sorted hash is like a cross between a hash and sorted set.
	// It uses a field name instead of the member for the purposes of identifying and
	// lexicographically sorting members.
	//
	// With DynamoDB, the field is limited to approximately 1024 bytes while the member is not.
	//
	// No conditionals are applied.
	ZHAdd(key, field string, member interface{}) AtomicWriteResult

	// Removes a member from a sorted hash. No conditionals are applied.
	ZHRem(key, field string) AtomicWriteResult

	// Adds a member to a set. No conditionals are applied.
	SAdd(key string, member interface{}, members ...interface{}) AtomicWriteResult

	// Removes a member from a set. No conditionals are applied.
	SRem(key string, member interface{}, members ...interface{}) AtomicWriteResult

	// Sets one or more fields of the hash at the given key. No conditionals are applied.
	HSet(key, field string, value interface{}, fields ...KeyValue) AtomicWriteResult

	// Sets one or more fields of the hash at the given key. The atomic write operation will be
	// aborted if the field already exists.
	HSetNX(key, field string, value interface{}) AtomicWriteResult

	// Deletes one or more fields of the hash at the given key. No conditionals are applied.
	HDel(key, field string, fields ...string) AtomicWriteResult

	// Executes the operation. If a condition failed, returns false.
	Exec() (bool, error)
}
