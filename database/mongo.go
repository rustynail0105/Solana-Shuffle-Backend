package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func InsertOne(collectionName string, document interface{}) error {
	collection := MDB.Collection(collectionName)

	_, err := collection.InsertOne(
		context.TODO(),
		document,
	)

	return err
}

func DeleteOne(collectionName string, filter bson.M) error {
	collection := MDB.Collection(collectionName)

	_, err := collection.DeleteOne(
		context.TODO(),
		filter,
	)

	return err
}

func UpdateOne(collectionName string, filter bson.M, update bson.M, upsert bool) error {
	collection := MDB.Collection(collectionName)

	_, err := collection.UpdateOne(
		context.TODO(),
		filter,
		update,
		options.Update().SetUpsert(upsert),
	)

	return err
}

func UpdateMany(collectionName string, filter bson.M, update bson.M) error {
	collection := MDB.Collection(collectionName)

	_, err := collection.UpdateMany(
		context.TODO(),
		filter,
		update,
	)

	return err
}

func ReplaceOne(collectionName string, filter bson.M, document interface{}) error {
	collection := MDB.Collection(collectionName)

	_, err := collection.ReplaceOne(
		context.TODO(),
		filter,
		document,
		options.Replace().SetUpsert(true),
	)

	return err
}

func FindOne(collectionName string, filter bson.M, res interface{}) error {
	collection := MDB.Collection(collectionName)

	return collection.FindOne(context.TODO(), filter).Decode(res)
}

func Find(collectionName string, filter bson.M, res interface{}) error {
	collection := MDB.Collection(collectionName)

	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())
	return cursor.All(context.TODO(), res)
}
