package db

import (
	"context"
	"log"
	"time"

	"github.com/dabeecao/testflight-checker/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	Client        *mongo.Client
	Users         *mongo.Collection
	Subscriptions *mongo.Collection
	AppStatus     *mongo.Collection
}

func Connect(cfg *config.Config) *Database {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	db := client.Database(cfg.MongoDBName)

	return &Database{
		Client:        client,
		Users:         db.Collection("users"),
		Subscriptions: db.Collection("subscriptions"),
		AppStatus:     db.Collection("app_status"),
	}
}

func (d *Database) CreateIndexes() {
	ctx := context.Background()
	
	// Index for subscriptions: search by user_id and tf_id
	d.Subscriptions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "tf_id", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "tf_id", Value: 1}}},
	})

	// Index for app_status: search by tf_id
	d.AppStatus.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "tf_id", Value: 1}},
	})
}

type Subscription struct {
	UserID int64  `bson:"user_id"`
	TFID   string `bson:"tf_id"`
	Title  string `bson:"title"`
}

type AppStatus struct {
	TFID          string `bson:"tf_id"`
	LastFreeSlots bool   `bson:"last_free_slots"`
}
