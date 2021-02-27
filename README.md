# keyvaluestore [![Documentation](https://godoc.org/github.com/ccbrown/keyvaluestore?status.svg)](https://godoc.org/github.com/ccbrown/keyvaluestore)

This package provides an interface with a Redis-like API and implementations for multiple key-value store backends. It supports profiling, caching, batching, eventual consistency, and transactions. It has implementations for:

* Redis, which is ideal for dev environments due to its low overhead.
* DynamoDB, which is ideal in production due to its incredible scalability.
* FoundationDB, which is ideal in production when you want control over the hardware or when DynamoDB's latency isn't good enough.
* An in-memory store, which is ideal for unit tests as it can be trivially destroyed and created.

This project originated at the AAF, where it was used in production until the company went bankrupt. I (@ccbrown) believe it's the best way to use DynamoDB in Go applications, so I'm continuing to maintain it and use it for other projects.

## Examples

Let's assume you have a struct representing your persistence layer like so:

```go
type Store struct {
    backend keyvaluestore.Backend
}
```

See the section below on backends for details on how to initialize the `backend` field.

### Storing an Object

```go
func (s *Store) AddUser(user *model.User) error {
    serialized, err := json.Marshal(user)
    if err != nil {
        return err
    }
    return s.backend.Set("user:" + string(user.Id), serialized)
}
```

This is the simplest way to store an object: Serialize it (JSON or [MessagePack](https://msgpack.org) works well), then use `Set` to store it. Alternatively, you could just implement [BinaryMarshaler](https://golang.org/pkg/encoding/#BinaryMarshaler) on your objects and skip the serialization step here.

### Getting an Object

Building off of the previous example, if you have a user's id, you can retrieve them like so:

```go
func (s *Store) GetUserById(id model.Id) (*model.User, error) {
    serialized, err := s.backend.Get("user:" + string(id))
    if serialized == nil {
        return nil, err
    }
    var user *model.User
    if err := json.Unmarshal([]byte(*serialized), &user); err != nil {
        return nil, err
    }
    return user, nil
}
```

### Storing an Object, Part 2

The first example has two big problems:

1. Users aren't accessible unless you have their id and there's no way to enumerate them.
2. It doesn't enforce uniqueness constraints for usernames or email addresses. (Let's assume all our users have usernames and email addresses.)

The first problem can be most easily solved with sorted sets: Simply add all users to a sorted set, which can be easily enumerated later. The second problem requires the use of transactions. In this case, the type of transaction we want is an atomic write operation.

```go
var ErrEmailAddressInUse = fmt.Errorf("email address in use")
var ErrUsernameInUse = fmt.Errorf("username in use")

func (s *Store) AddUser(user *model.User) error {
    serialized, err := json.Marshal(user)
    if err != nil {
        return err
    }

    tx := s.backend.AtomicWrite()
    tx.Set("user:" + string(user.Id), serialized)
    tx.ZHAdd("usernames", user.Username, user.Id, 0.0)
    usernameSet := tx.SetNX("user_by_username:"+user.Username, user.Id)
    tx.SetNX("user_by_email_address:"+user.EmailAddress, user.Id)

    if didCommit, err := tx.Exec(); err != nil {
        return err
    } else if didCommit {
        return nil
    } else if usernameSet.ConditionalFailed() {
        return ErrUsernameInUse
    }
    return ErrEmailAddressInUse
}
```

This implementation now covers all the bases:

* We can enumerate users, sorted by username, by iterating over the "usernames" set.
* If the username is already taken, the transaction will be aborted and the function will return `ErrUsernameInUse`.
* If the email address is already taken, the transaction will be aborted and the function will return `ErrEmailAddressInUse`.
* We also have the ability to look up users by their username or email address.

### Getting Multiple Objects

In many scenarios, you'll want to fetch more than one user at once. If you made one round-trip to the backend per user, this would be very slow. To efficiently fetch multiple objects or perform multiple operations, you can use batching:

```go
func (s *Store) GetUsersByIds(ids ...model.Id) ([]*model.User, error) {
    batch := s.backend.Batch()
    gets := make([]keyvaluestore.GetResult, len(ids))
    for i, id := range ids {
        gets[i] = batch.Get("user:" + string(id))
    }
    if err := batch.Exec(); err != nil {
        return nil, err
    }

    users := make([]*model.User, 0, len(gets))
    for _, get := range gets {
        if v, _ := get.Result(); v != nil {
            var user *model.User
            if err := json.Unmarshal([]byte(*v), &user); err != nil {
                return nil, err
            }
            users = append(users, user)
        }
    }
    return users, nil
}
```

## Backends

### Memory

In-memory backends are ideal for unit tests as you can create and destroy thousands of them quickly and cheaply:

```go
backend := memorystore.NewBackend()
backend.Set("foo", "bar")
```

### Redis

Redis backends are ideal for dev environments as they're lightweight and easy to spin up and tear down:

```go
backend := &redisstore.Backend{
    Client: redis.NewClient(&redis.Options{
        Addr: "127.0.0.1:6379",
    }),
}
backend.Set("foo", "bar")
```

### DynamoDB

DynamoDB is ideal for production in AWS as it's easy to set up and maintain and scales incredibly well.

When you create your DynamoDB table, you'll need to give it the following schema:

```yaml
DynamoDBTable:
  Type: AWS::DynamoDB::Table
  Properties:
    AttributeDefinitions:
      - AttributeName: hk
        AttributeType: B
      - AttributeName: rk
        AttributeType: B
      - AttributeName: rk2
        AttributeType: B
    KeySchema:
      - AttributeName: hk
        KeyType: HASH
      - AttributeName: rk
        KeyType: RANGE
    LocalSecondaryIndexes:
      - IndexName: rk2
        KeySchema:
          - AttributeName: hk
            KeyType: HASH
          - AttributeName: rk2
            KeyType: RANGE
        Projection:
          ProjectionType: ALL
    BillingMode: PAY_PER_REQUEST
```

If you're using CloudFormation, you can just copy/paste that into your template.

Then you can connect to it like so:

```go
session := session.Must(session.NewSession())
backend := &KeyValueStore{
    backend: &dynamodbstore.Backend{
        Client: dynamodb.New(session),
        TableName: cfg.TableName,
    },
}
backend.Set("foo", "bar")
```

You can also create the backend using a DAX client for improved performance.
