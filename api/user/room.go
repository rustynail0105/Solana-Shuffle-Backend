package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/utility"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleGetRoomGET(c *fiber.Ctx) error {
	roomID := c.Params("+")

	room, err := shuffle.GetRoom(roomID)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(room)
}

func HandleRoomHistoryGET(c *fiber.Ctx) error {
	roomID := c.Params("+")

	sessions := []shuffle.Session{}
	cursor, err := database.MDB.Collection("sessions").Find(
		context.TODO(),
		bson.M{"roomId": roomID},
		options.Find().SetSort(bson.M{"closeTime": -1}).SetLimit(20),
	)
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}
	defer cursor.Close(context.TODO())
	err = cursor.All(context.TODO(), &sessions)
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}

	var amtRemove int
	for i, s := range sessions {
		if s.CloseTime == 0 {
			sessions = utility.Remove(sessions, i-amtRemove)
			amtRemove++
		}
	}

	return c.Status(fiber.StatusOK).JSON(sessions)
}

func HandleRoomUserCountGET(c *fiber.Ctx) error {
	// Calculate the time 10 minutes ago
	tenMinutesAgo := time.Now().Add(-10 * time.Minute).Unix()

	// Define the filter to match sessions that have been closed within the last 10 minutes
	filter := bson.M{
		"closeTime": bson.M{
			"$gt": tenMinutesAgo,
		},
	}

	// Define the projection to include only the necessary fields for the aggregation pipeline
	projection := bson.M{
		"roomId":          1,
		"users.publickey": 1,
	}

	// Define the aggregation pipeline to count distinct users by room
	pipeline := mongo.Pipeline{
		// Match sessions closed within the last 10 minutes
		{{"$match", filter}},
		// Unwind the users array
		{{"$unwind", "$users"}},
		// Project only the necessary fields
		{{"$project", projection}},
		// Group by room and collect unique user public keys within each room
		{{"$group", bson.M{
			"_id": "$roomId",
			"publickeys": bson.M{
				"$addToSet": "$users.publickey",
			},
		}}},
		// Project the room ID and count the number of unique user public keys
		{{"$project", bson.M{
			"count": bson.M{
				"$size": "$publickeys",
			},
		}}},
	}

	// Execute the aggregation pipeline and collect the results
	cursor, err := database.MDB.Collection("sessions").Aggregate(context.Background(), pipeline)
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}
	defer cursor.Close(context.Background())

	var results []bson.M
	if err := cursor.All(context.Background(), &results); err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}

	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}
	return c.Status(fiber.StatusOK).JSON(results)
}

type createRoomBody struct {
	Signature             solana.Signature `json:"signature"`
	Name                  string           `json:"name"`
	PublicKey             solana.PublicKey `json:"publicKey"`
	Public                bool             `json:"public"`
	CreatorFeeBasisPoints int              `json:"creatorFeeBasisPoints"`
	TokenTicker           string           `json:"tokenTicker"`

	MinimumAmount int `json:"minimumAmount"`
	MaximumAmount int `json:"maximumAmount"`
}

func HandleCreateRoomPOST(c *fiber.Ctx) error {
	body := new(createRoomBody)
	err := c.BodyParser(body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.Name == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("name missing"))
	}
	if body.CreatorFeeBasisPoints < 0 {
		return JSONError(c, fiber.StatusBadRequest, errors.New("creator fees cannot be negative"))
	}
	if body.CreatorFeeBasisPoints > 500 {
		return JSONError(c, fiber.StatusBadRequest, errors.New("creator fees cannot be higher than 500"))
	}
	if body.TokenTicker == "" {
		body.TokenTicker = "SOL"
	}
	var token database.Token
	err = database.FindOne("tokens", bson.M{"ticker": body.TokenTicker}, &token)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid token"))
	}

	wanted := []byte(fmt.Sprintf(
		"solanashuffle create room %s", body.PublicKey.String(),
	))

	if !vsolana.VerifySignature(body.Signature, body.PublicKey, wanted) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	if body.PublicKey != env.House().PublicKey() {
		user := database.NewUser(body.PublicKey)
		err = database.FindOne("users", bson.M{"publicKey": body.PublicKey}, &user)
		if err != nil || user.Stats.TotalVolume < solana.LAMPORTS_PER_SOL*1 {
			return JSONError(c, fiber.StatusBadRequest, fmt.Errorf("user must bet at least 1 SOL to create a room. Current amount: %f", float64(user.Stats.TotalVolume)/float64(solana.LAMPORTS_PER_SOL)))
		}
	}

	var nameRooms shuffle.Room
	err = database.FindOne("rooms", bson.M{"name": body.Name}, &nameRooms)
	if err == nil {
		return JSONError(c, fiber.StatusBadRequest, errors.New("room with that name already exists"))
	}

	if body.MinimumAmount > body.MaximumAmount {
		return JSONError(c, fiber.StatusBadRequest, errors.New("minimum amount must be smaller than maximum amount"))
	}

	room, err := shuffle.NewRoom(
		shuffle.CreateRoomConfig{
			Creator:               body.PublicKey,
			CreatorFeeBasisPoints: body.CreatorFeeBasisPoints,
			Name:                  body.Name,
			Public:                body.Public,
			Token:                 token,

			MinimumAmount: body.MinimumAmount,
			MaximumAmount: body.MaximumAmount,

			Official: body.PublicKey == env.House().PublicKey(),
		},
	)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(room)
}

type favoriteRoomBody struct {
	PublicKey solana.PublicKey `json:"publicKey"`
	Signature solana.Signature `json:"signature"`
}

func HandleFavoriteRoomPOST(c *fiber.Ctx) error {
	roomID := c.Params("+")

	body := new(favoriteRoomBody)
	err := c.BodyParser(body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	wanted := []byte(fmt.Sprintf(
		"solanashuffle favorite room %s %s", roomID, body.PublicKey.String(),
	))

	if !vsolana.VerifySignature(body.Signature, body.PublicKey, wanted) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	_, err = shuffle.GetRoom(roomID)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, errors.New("room not found"))
	}

	user := database.NewUser(body.PublicKey)
	err = database.FindOne("users", bson.M{"publicKey": body.PublicKey}, &user)
	if err == nil {
		for _, fav := range user.FavoriteRooms {
			if fav == roomID {
				return JSONError(c, fiber.StatusBadRequest, errors.New("room already favorited"))
			}
		}
	}

	database.UpdateOne(
		"users",
		bson.M{"publicKey": body.PublicKey},
		bson.M{"$push": bson.M{"favoriteRooms": roomID}},
		true,
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "success",
	})
}

func HandleUnfavoriteRoomDELETE(c *fiber.Ctx) error {
	roomID := c.Params("+")

	body := new(favoriteRoomBody)
	err := c.BodyParser(body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	wanted := []byte(fmt.Sprintf(
		"solanashuffle unfavorite room %s %s", roomID, body.PublicKey.String(),
	))

	if !vsolana.VerifySignature(body.Signature, body.PublicKey, wanted) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	_, err = shuffle.GetRoom(roomID)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, errors.New("room not found"))
	}

	database.UpdateOne(
		"users",
		bson.M{"publicKey": body.PublicKey},
		bson.M{"$pull": bson.M{"favoriteRooms": roomID}},
		true,
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "success",
	})
}
