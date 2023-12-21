package user

import (
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleTokensGET(c *fiber.Ctx) error {
	var tokens []database.Token
	err := database.Find(
		"tokens",
		bson.M{},
		&tokens,
	)
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}

	return c.Status(fiber.StatusOK).JSON(tokens)
}
