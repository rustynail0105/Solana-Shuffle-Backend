package user

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/utility"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProfileBody database.UserProfile

// HandleProfileHistoryGET Get User play history
func HandleProfileHistoryGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	page, err := strconv.Atoi(c.FormValue("page"))
	if err != nil {
		page = 0
	}
	page++

	limit := int64(20)
	skip := int64(int64(page)*limit - limit)

	sessions := []shuffle.Session{}
	cursor, err := database.MDB.Collection("sessions").Find(
		context.TODO(),
		bson.M{"users.publickey": publicKey},
		options.Find().SetSort(bson.M{"closeTime": -1}).SetLimit(limit).SetSkip(skip),
	)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
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

	rooms := make(map[string]shuffle.Room)

	for _, s := range sessions {
		r, err := shuffle.GetRoom(s.RoomID)
		if err != nil {
			continue
		}
		rooms[s.RoomID] = *r
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"sessions": sessions,
		"rooms":    rooms,
	})
}

// HandleProfileNameSET Set Name
func HandleProfileNameSET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(ProfileBody)
	err = c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.Name == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid name"))
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"name": body.Name}},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(body.Name)
}

// HandleProfileImageSET Set Image
func HandleProfileImageSET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(ProfileBody)
	err = c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.Image == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid image"))
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"image": body.Image}},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(body.Image)
}

// HandleProfileBannerSET Set Banner
func HandleProfileBannerSET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(ProfileBody)
	err = c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.Banner == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid banner"))
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"banner": body.Banner}},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(body.Banner)
}

// HandleAboutSET Set About
func HandleAboutSET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(ProfileBody)
	err = c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if body.About == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid about"))
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"about": body.About}},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(body.About)
}

// HandleProfileGet Fetch Entire Profile
func HandleProfileGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	user := database.NewUser(publicKey)
	if err := database.FindOne(
		"users",
		bson.M{"publicKey": publicKey},
		&user,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}
