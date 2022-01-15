package mongo

import (
	"context"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoInst struct {
	client *mongo.Client
	db     *mongo.Database
}

func (i *mongoInst) Collection(name instance.MongoCollectionName) *mongo.Collection {
	return i.db.Collection(string(name))
}

func (i *mongoInst) Ping(ctx context.Context) error {
	return i.db.Client().Ping(ctx, nil)
}

func (i *mongoInst) RawClient() *mongo.Client {
	return i.client
}

func (i *mongoInst) RawDatabase() *mongo.Database {
	return i.db
}
