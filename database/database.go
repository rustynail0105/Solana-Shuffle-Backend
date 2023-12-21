package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/gagliardetto/solana-go"
	"github.com/go-redis/redis/v8"
	"github.com/solanashuffle/backend/env"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MDB *mongo.Database
	RDB *redis.Client
)

// kHzJSq5F9WfEVHXI+YT4N5E4nDC9WXje1aU/X0UForBL5MFdSHhudkmZr4sK00aldSS9Fxkj5/gY3g5S2kHPrg==

func ConnectDatabases() {
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(env.GetDatabaseURL()).
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	MDB = client.Database("home")

	RDB = redis.NewClient(&redis.Options{
		Addr:     "52.70.17.114:6379",
		Password: "kHzJSq5F9WfEVHXI+YT4N5E4nDC9WXje1aU/X0UForBL5MFdSHhudkmZr4sK00aldSS9Fxkj5/gY3g5S2kHPrg==",
	})
}

func GetUser(publicKey solana.PublicKey) (UserProfile, error) {
	key := fmt.Sprintf("user:%s", publicKey.String())

	val, err := RDB.Get(
		context.TODO(),
		key,
	).Result()
	if err != nil {
		return UserProfile{}, err
	}

	var user UserProfile
	err = json.Unmarshal([]byte(val), &user)
	return user, err
}

func DbGetUser(publicKey solana.PublicKey) (UserProfile, error) {
	user := NewUser(publicKey)
	err := FindOne(
		"users",
		bson.M{"publicKey": publicKey},
		&user,
	)
	if err != nil {
		return UserProfile{}, err
	}

	return user, nil
}

func NewUser(publicKey solana.PublicKey) UserProfile {
	return UserProfile{
		PublicKey: publicKey,
		Name:      publicKey.Short(4),
		Image:     "",

		FavoriteRooms: []string{},

		Stats: Stats{
			Volumes: make(map[string]uint64),
			Games:   make(map[string]uint64),
		},
	}
}
