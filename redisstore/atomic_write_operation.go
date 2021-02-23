package redisstore

import (
	"fmt"
	"strings"

	"github.com/go-redis/redis"

	"github.com/ccbrown/keyvaluestore"
)

type AtomicWriteOperation struct {
	Client *redis.Client

	operations []*atomicWriteOperation
}

type atomicWriteOperation struct {
	keys      []string
	condition string
	write     string
	args      []interface{}

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
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('set', @0, $0)",
		args:      []interface{}{value},
	})
}

func (op *AtomicWriteOperation) SetNX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('exists', @0) == 0",
		write:     "redis.call('set', @0, $0)",
		args:      []interface{}{value},
	})
}

func (op *AtomicWriteOperation) SetXX(key string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('exists', @0) == 1",
		write:     "redis.call('set', @0, $0)",
		args:      []interface{}{value},
	})
}

func (op *AtomicWriteOperation) SetEQ(key string, value, oldValue interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('get', @0) == $0",
		write:     "redis.call('set', @0, $1)",
		args:      []interface{}{oldValue, value},
	})
}

func (op *AtomicWriteOperation) Delete(key string) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('del', @0)",
	})
}

func (op *AtomicWriteOperation) DeleteXX(key string) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('exists', @0) == 1",
		write:     "redis.call('del', @0)",
	})
}

func (op *AtomicWriteOperation) NIncrBy(key string, n int64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('incrby', @0, $0)",
		args:      []interface{}{n},
	})
}

func (op *AtomicWriteOperation) ZAdd(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('zadd', @0, $1, $0)",
		args:      []interface{}{member, score},
	})
}

func (op *AtomicWriteOperation) ZHAdd(key, field string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key, zhHashKey(key)},
		condition: "true",
		write:     "redis.call('zadd', @0, $1, $0)\nredis.call('hset', @1, $0, $2)",
		args:      []interface{}{field, score, member},
	})
}

func (op *AtomicWriteOperation) ZAddNX(key string, member interface{}, score float64) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('zscore', @0, $0) == false",
		write:     "redis.call('zadd', @0, $1, $0)",
		args:      []interface{}{member, score},
	})
}

func (op *AtomicWriteOperation) ZRem(key string, member interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('zrem', @0, $0)",
		args:      []interface{}{member},
	})
}

func (op *AtomicWriteOperation) ZHRem(key, field string) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key, zhHashKey(key)},
		condition: "true",
		write:     "redis.call('zrem', @0, $0)\nredis.call('hdel', @1, $0)",
		args:      []interface{}{field},
	})
}

func (op *AtomicWriteOperation) SAdd(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	placeholders := make([]string, 1+len(members))
	for i := 0; i < len(placeholders); i++ {
		placeholders[i] = fmt.Sprintf("$%v", i)
	}
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('sadd', @0, " + strings.Join(placeholders, ", ") + ")",
		args:      append([]interface{}{member}, members...),
	})
}

func (op *AtomicWriteOperation) SRem(key string, member interface{}, members ...interface{}) keyvaluestore.AtomicWriteResult {
	placeholders := make([]string, 1+len(members))
	for i := 0; i < len(placeholders); i++ {
		placeholders[i] = fmt.Sprintf("$%v", i)
	}
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('srem', @0, " + strings.Join(placeholders, ", ") + ")",
		args:      append([]interface{}{member}, members...),
	})
}

func (op *AtomicWriteOperation) HSet(key, field string, value interface{}, fields ...keyvaluestore.KeyValue) keyvaluestore.AtomicWriteResult {
	placeholders := make([]string, 2*(len(fields)+1))
	for i := 0; i < len(placeholders); i++ {
		placeholders[i] = fmt.Sprintf("$%v", i)
	}
	args := make([]interface{}, 0, 2*(len(fields)+1))
	args = append(args, field)
	args = append(args, value)
	for _, field := range fields {
		args = append(args, field.Key)
		args = append(args, field.Value)
	}
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('hset', @0, " + strings.Join(placeholders, ", ") + ")",
		args:      args,
	})
}

func (op *AtomicWriteOperation) HSetNX(key, field string, value interface{}) keyvaluestore.AtomicWriteResult {
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "redis.call('hexists', @0, $0) == 0",
		write:     "redis.call('hset', @0, $0, $1)",
		args:      []interface{}{field, value},
	})
}

func (op *AtomicWriteOperation) HDel(key string, field string, fields ...string) keyvaluestore.AtomicWriteResult {
	placeholders := make([]string, 1+len(fields))
	for i := 0; i < len(placeholders); i++ {
		placeholders[i] = fmt.Sprintf("$%v", i)
	}
	args := make([]interface{}, 0, len(fields)+1)
	args = append(args, field)
	for _, field := range fields {
		args = append(args, field)
	}
	return op.write(&atomicWriteOperation{
		keys:      []string{key},
		condition: "true",
		write:     "redis.call('hdel', @0, " + strings.Join(placeholders, ", ") + ")",
		args:      args,
	})
}

func preprocessAtomicWriteExpression(in string, keysOffset, numKeys int, argsOffset, numArgs int) string {
	out := in
	for i := numKeys - 1; i >= 0; i-- {
		out = strings.Replace(out, fmt.Sprintf("@%d", i), fmt.Sprintf("KEYS[%d]", keysOffset+i+1), -1)
	}
	for i := numArgs - 1; i >= 0; i-- {
		out = strings.Replace(out, fmt.Sprintf("$%d", i), fmt.Sprintf("ARGV[%d]", argsOffset+i+1), -1)
	}
	return out
}

func (op *AtomicWriteOperation) Exec() (bool, error) {
	if len(op.operations) > keyvaluestore.MaxAtomicWriteOperations {
		return false, fmt.Errorf("max operation count exceeded")
	}

	var keys []string
	var args []interface{}
	writeExpressions := make([]string, len(op.operations))

	script := []string{"local checks = {}"}
	for i, op := range op.operations {
		script = append(script, fmt.Sprintf("checks[%d] = %s", i+1, preprocessAtomicWriteExpression(op.condition, len(keys), len(op.keys), len(args), len(op.args))))
		writeExpressions[i] = preprocessAtomicWriteExpression(op.write, len(keys), len(op.keys), len(args), len(op.args))
		keys = append(keys, op.keys...)
		args = append(args, op.args...)
	}
	script = append(script,
		"for i, v in ipairs(checks) do",
		"if not v then",
		"return checks",
		"end",
		"end",
	)
	script = append(script, writeExpressions...)
	script = append(script,
		"return checks",
	)

	result, err := op.Client.Eval(strings.Join(script, "\n"), keys, args...).Result()
	if err != nil {
		return false, err
	}

	checks, ok := result.([]interface{})
	if !ok {
		return false, fmt.Errorf("unexpected return type: %T", result)
	} else if len(checks) != len(op.operations) {
		return false, fmt.Errorf("not enough return values")
	}

	ret := true
	for i, check := range checks {
		if check != nil {
			op.operations[i].conditionPassed = true
		} else {
			ret = false
		}
	}
	return ret, nil
}
