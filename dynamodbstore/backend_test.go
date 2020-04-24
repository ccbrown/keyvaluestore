package dynamodbstore

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	"github.com/ccbrown/keyvaluestore"
	"github.com/ccbrown/keyvaluestore/keyvaluestoretest"
)

func newDynamoDBTestConfig() (*aws.Config, error) {
	endpoint := os.Getenv("DYNAMODB_ENDPOINT")

	keyBytes := make([]byte, 20)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, err
	}
	key := base64.RawURLEncoding.EncodeToString(keyBytes)

	config := &aws.Config{
		Region: aws.String("us-east-1"),
		EndpointResolver: endpoints.ResolverFunc(func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
			if endpoint != "" {
				return endpoints.ResolvedEndpoint{
					URL: endpoint,
				}, nil
			}
			return endpoints.ResolvedEndpoint{
				URL: "http://localhost:8000",
			}, nil
		}),
		Credentials: credentials.NewStaticCredentials(key, key, ""),
		MaxRetries:  aws.Int(0),
	}

	client := dynamodb.New(session.Must(session.NewSession(config)))
	if endpoint == "" {
		if _, err := client.ListTables(&dynamodb.ListTablesInput{}); err != nil {
			if err, ok := err.(awserr.Error); ok && err.Code() == "RequestError" {
				return nil, nil
			}
		}
	}
	return config, nil
}

func newDynamoDBTestClient() (*dynamodb.DynamoDB, error) {
	config, err := newDynamoDBTestConfig()
	if config == nil {
		return nil, err
	}
	return dynamodb.New(session.Must(session.NewSession(config))), nil
}

func recreateTable(client *dynamodb.DynamoDB, tableName string) error {
	if _, err := client.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	}); err == nil {
		client.WaitUntilTableNotExists(&dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})
	}

	return CreateDefaultTable(client, tableName)
}

func newTestBackend(client *dynamodb.DynamoDB, tableName string) *Backend {
	if err := recreateTable(client, tableName); err != nil {
		panic(err)
	}

	return &Backend{
		Client:    client,
		TableName: tableName,
	}
}

func TestBackend(t *testing.T) {
	client, err := newDynamoDBTestClient()
	if err != nil {
		t.Fatal(err)
	} else if client == nil {
		t.Skip("no dynamodb server available. to start one: docker run -p 8000:8000 --rm -it amazon/dynamodb-local")
	}

	keyvaluestoretest.TestBackend(t, func() keyvaluestore.Backend {
		return newTestBackend(client, "TestBackend")
	})
}
