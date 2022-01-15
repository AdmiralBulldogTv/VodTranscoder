package instance

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type Mongo interface {
	Collection(MongoCollectionName) *mongo.Collection
	Ping(ctx context.Context) error
	RawClient() *mongo.Client
	RawDatabase() *mongo.Database
}

type MongoCollectionName string
