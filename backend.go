package keyvaluestore

type KeyValue struct {
	Key   string
	Value interface{}
}

type Backend interface {
	// Batch allows you to batch up simple operations for better performance potential. Use this
	// only for possible performance benefits. Read isolation is implementation-defined and other
	// properties such as atomicity should not be assumed.
	Batch() BatchOperation

	// AtomicWrite executes up to 25 write operations atomically, failing entirely if any
	// conditional operations (e.g. SetNX) are not executed.
	AtomicWrite() AtomicWriteOperation

	Delete(key string) (success bool, err error)
	Get(key string) (*string, error)
	Set(key string, value interface{}) error

	// Set if the key already exists.
	SetXX(key string, value interface{}) (bool, error)

	// Set if the key doesn't exist.
	SetNX(key string, value interface{}) (bool, error)

	// Set if the key exists and its value is equal to the given one.
	SetEQ(key string, value, oldValue interface{}) (success bool, err error)

	// Increments the number with the given key by some number. If the key doesn't exist, it's set
	// to the given number instead. To get the current value, you can pass 0 as n.
	NIncrBy(key string, n int64) (int64, error)

	// Add to or create a set. Sets are ideal for small sizes, but have implementation-dependent
	// size limitations (400KB for DynamoDB). For large or unbounded sets, use ZAdd instead.
	SAdd(key string, member interface{}, members ...interface{}) error

	// Remove from a set.
	SRem(key string, member interface{}, members ...interface{}) error

	// Get members of a set.
	SMembers(key string) ([]string, error)

	// Sets one or more fields of the hash at the given key. If no hash exists at the key, a new one
	// is created. Hashes are ideal for small sizes, but have implementation-dependent size
	// limitations (400KB for DynamoDB). For large or unbounded sets, use something else.
	HSet(key, field string, value interface{}, fields ...KeyValue) error

	// Deletes one or more fields of the hash at the given key.
	HDel(key, field string, fields ...string) error

	// Gets a field of the hash at the given key or nil if the hash or field does not exist.
	HGet(key, field string) (*string, error)

	// Gets all fields of the hash at the given key.
	HGetAll(key string) (map[string]string, error)

	// Add to or create a sorted set. The size of the member may be limited by some backends (for
	// example, DynamoDB limits it to approximately 1024 bytes).
	ZAdd(key string, member interface{}, score float64) error

	// Gets the score for a member added via ZAdd.
	ZScore(key string, member interface{}) (*float64, error)

	// Remove from a sorted set.
	ZRem(key string, member interface{}) error

	// Increment a score in a sorted set or set the score if the member doesn't exist.
	ZIncrBy(key string, member string, n float64) (float64, error)

	// Get members of a sorted set by ascending score.
	ZRangeByScore(key string, min, max float64, limit int) ([]string, error)

	// Get members (and their scores) of a sorted set by ascending score.
	ZRangeByScoreWithScores(key string, min, max float64, limit int) (ScoredMembers, error)

	// Get members of a sorted set by descending score.
	ZRevRangeByScore(key string, min, max float64, limit int) ([]string, error)

	// Get members (and their scores) of a sorted set by descending score.
	ZRevRangeByScoreWithScores(key string, min, max float64, limit int) (ScoredMembers, error)

	// Gets the number of members with scores between min and max, inclusive. This method can get
	// somewhat expensive on DynamoDB as it is not a constant-time operation.
	ZCount(key string, min, max float64) (int, error)

	// Gets the number of members between min and max. All members of the set must have been added
	// with a zero score. min and max must begin with '(' or '[' to indicate exclusive or inclusive.
	// Alternatively, min can be "-" and max can be "+" to represent infinities. This method can get
	// somewhat expensive on DynamoDB as it is not a constant-time operation.
	ZLexCount(key string, min, max string) (int, error)

	// Get members of a sorted set by lexicographical order. All members of the set must have been
	// added with a zero score. min and max must begin with '(' or '[' to indicate exclusive or
	// inclusive. Alternatively, min can be "-" and max can be "+" to represent infinities.
	ZRangeByLex(key string, min, max string, limit int) ([]string, error)

	// Get members of a sorted set by reverse lexicographical order. All members of the set must
	// have been added with a zero score. min and max must begin with '(' or '[' to indicate
	// exclusive or inclusive. Alternatively, min can be "-" and max can be "+" to represent
	// infinities.
	ZRevRangeByLex(key string, min, max string, limit int) ([]string, error)

	// Add to or create a sorted hash. A sorted hash is like a cross between a hash and sorted set.
	// It uses a field name instead of the member for the purposes of identifying and
	// lexicographically sorting members.
	//
	// With DynamoDB, the field is limited to approximately 1024 bytes while the member is not.
	ZHAdd(key, field string, member interface{}, score float64) error

	// Remove from a sorted hash.
	ZHRem(key, field string) error

	// Get members of a sorted hash by ascending score.
	ZHRangeByScore(key string, min, max float64, limit int) ([]string, error)

	// Get members (and their scores) of a sorted hash by ascending score.
	ZHRangeByScoreWithScores(key string, min, max float64, limit int) (ScoredMembers, error)

	// Get members of a sorted hash by descending score.
	ZHRevRangeByScore(key string, min, max float64, limit int) ([]string, error)

	// Get members (and their scores) of a sorted hash by descending score.
	ZHRevRangeByScoreWithScores(key string, min, max float64, limit int) (ScoredMembers, error)

	// Get members of a sorted hash by their fields' lexicographical order. All members of the set
	// must have been added with a zero score. min and max must begin with '(' or '[' to indicate
	// exclusive or inclusive. Alternatively, min can be "-" and max can be "+" to represent
	// infinities.
	ZHRangeByLex(key string, min, max string, limit int) ([]string, error)

	// Get members of a sorted hash by their fields' reverse lexicographical order. All members of
	// the set must have been added with a zero score. min and max must begin with '(' or '[' to
	// indicate exclusive or inclusive. Alternatively, min can be "-" and max can be "+" to
	// represent infinities.
	ZHRevRangeByLex(key string, min, max string, limit int) ([]string, error)

	// For performance improvements you may wish to enable eventually consistent reads for backends
	// that support it.
	WithEventuallyConsistentReads() Backend

	// Some backends support metrics via profilers. See the Profiler interfaces in the specific
	// implementation packages.
	WithProfiler(profiler interface{}) Backend

	// If the backend wraps another (e.g. a read cache that wraps a redis backend), this returns the
	// wrapped backend.
	Unwrap() Backend
}

type ScoredMembers []*ScoredMember

func (m ScoredMembers) Values() []string {
	result := make([]string, len(m))

	for i, member := range m {
		result[i] = member.Value
	}

	return result
}

type ScoredMember struct {
	Score float64
	Value string
}
