package dynamodbstore

import (
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type Profiler interface {
	ConsumeDynamoDBReadCapacity(capacity float64)
	ConsumeDynamoDBWriteCapacity(capacity float64)
	AddDynamoDBRequestProfile(operationName string, duration time.Duration)
}

type BasicProfiler struct {
	requestCount            int64
	requestNanoseconds      int64
	readCapacityConsumedX4  int64
	writeCapacityConsumedX4 int64
}

func (p *BasicProfiler) ConsumeDynamoDBReadCapacity(capacity float64) {
	atomic.AddInt64(&p.readCapacityConsumedX4, int64(capacity*4))
}

func (p *BasicProfiler) ConsumeDynamoDBWriteCapacity(capacity float64) {
	atomic.AddInt64(&p.writeCapacityConsumedX4, int64(capacity*4))
}

func (p *BasicProfiler) AddDynamoDBRequestProfile(operationName string, duration time.Duration) {
	atomic.AddInt64(&p.requestCount, 1)
	atomic.AddInt64(&p.requestNanoseconds, int64(duration/time.Nanosecond))
}

func (p *BasicProfiler) DynamoDBRequestCount() int {
	return int(atomic.LoadInt64(&p.requestCount))
}

func (p *BasicProfiler) DynamoDBRequestDuration() time.Duration {
	return time.Duration(atomic.LoadInt64(&p.requestNanoseconds)) * time.Nanosecond
}

func (p *BasicProfiler) DynamoDBReadCapacityConsumed() float64 {
	return float64(atomic.LoadInt64(&p.readCapacityConsumedX4)) / 4.0
}

func (p *BasicProfiler) DynamoDBWriteCapacityConsumed() float64 {
	return float64(atomic.LoadInt64(&p.writeCapacityConsumedX4)) / 4.0
}

type ProfilingBackendClient struct {
	Client   BackendClient
	Profiler Profiler
}

func (c *ProfilingBackendClient) profileConsumedReadCapacity(capacity *dynamodb.ConsumedCapacity) {
	if capacity == nil || capacity.CapacityUnits == nil {
		return
	}
	c.Profiler.ConsumeDynamoDBReadCapacity(*capacity.CapacityUnits)
}

func (c *ProfilingBackendClient) profileConsumedWriteCapacity(capacity *dynamodb.ConsumedCapacity) {
	if capacity == nil || capacity.CapacityUnits == nil {
		return
	}
	c.Profiler.ConsumeDynamoDBWriteCapacity(*capacity.CapacityUnits)
}

func (c *ProfilingBackendClient) BatchGetItem(input *dynamodb.BatchGetItemInput) (*dynamodb.BatchGetItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.BatchGetItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("BatchGetItem", time.Since(startTime))
	if err == nil {
		for _, capacity := range output.ConsumedCapacity {
			c.profileConsumedReadCapacity(capacity)
		}
	}
	return output, err
}

func (c *ProfilingBackendClient) BatchWriteItem(input *dynamodb.BatchWriteItemInput) (*dynamodb.BatchWriteItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.BatchWriteItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("BatchWriteItem", time.Since(startTime))
	if err == nil {
		for _, capacity := range output.ConsumedCapacity {
			c.profileConsumedWriteCapacity(capacity)
		}
	}
	return output, err
}

func (c *ProfilingBackendClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.DeleteItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("DeleteItem", time.Since(startTime))
	if err == nil {
		c.profileConsumedWriteCapacity(output.ConsumedCapacity)
	}
	return output, err
}

func (c *ProfilingBackendClient) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.GetItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("GetItem", time.Since(startTime))
	if err == nil {
		c.profileConsumedReadCapacity(output.ConsumedCapacity)
	}
	return output, err
}

func (c *ProfilingBackendClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.PutItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("PutItem", time.Since(startTime))
	if err == nil {
		c.profileConsumedWriteCapacity(output.ConsumedCapacity)
	}
	return output, err
}

func (c *ProfilingBackendClient) Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.Query(&copy)
	c.Profiler.AddDynamoDBRequestProfile("Query", time.Since(startTime))
	if err == nil {
		c.profileConsumedReadCapacity(output.ConsumedCapacity)
	}
	return output, err
}

func (c *ProfilingBackendClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.UpdateItem(&copy)
	c.Profiler.AddDynamoDBRequestProfile("UpdateItem", time.Since(startTime))
	if err == nil {
		c.profileConsumedWriteCapacity(output.ConsumedCapacity)
	}
	return output, err
}

func (c *ProfilingBackendClient) TransactWriteItems(input *dynamodb.TransactWriteItemsInput) (*dynamodb.TransactWriteItemsOutput, error) {
	copy := *input
	copy.ReturnConsumedCapacity = aws.String(dynamodb.ReturnConsumedCapacityTotal)
	startTime := time.Now()
	output, err := c.Client.TransactWriteItems(&copy)
	c.Profiler.AddDynamoDBRequestProfile("TransactWriteItem", time.Since(startTime))
	if err == nil {
		for _, capacity := range output.ConsumedCapacity {
			c.profileConsumedWriteCapacity(capacity)
		}
	}
	return output, err
}
