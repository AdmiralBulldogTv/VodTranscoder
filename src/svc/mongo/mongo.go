package mongo

import (
	"context"

	"github.com/AdmiralBulldogTv/VodTranscoder/src/instance"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func New(ctx context.Context, opt SetupOptions) (instance.Mongo, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(opt.URI).SetDirect(opt.Direct))
	if err != nil {
		return nil, err
	}

	// Send a Ping
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}

	database := client.Database(opt.Database)

	for _, ind := range opt.Indexes {
		col := ind.Collection
		if name, err := database.Collection(string(col)).Indexes().CreateOne(ctx, ind.Index); err != nil {
			panic(err)
		} else {
			logrus.WithField("collection", col).Infof("Collection index created: %s", name)
		}
	}

	logrus.Info("mongo, ok")

	return &mongoInst{
		client: client,
		db:     database,
	}, nil
}

type SetupOptions struct {
	URI      string
	Database string
	Direct   bool
	Indexes  []IndexRef
}

type IndexRef struct {
	Collection instance.MongoCollectionName
	Index      mongo.IndexModel
}

var (
	ErrNoDocuments = mongo.ErrNoDocuments
)

const (
	CollectionUsers   instance.MongoCollectionName = "users"
	CollectionStreams instance.MongoCollectionName = "streams"
)

type (
	Pipeline       = mongo.Pipeline
	WriteModel     = mongo.WriteModel
	InsertOneModel = mongo.InsertOneModel
	UpdateOneModel = mongo.UpdateOneModel
	IndexModel     = mongo.IndexModel
)
