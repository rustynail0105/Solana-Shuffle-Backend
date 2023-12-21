package user

import (
	"errors"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/tower"
	"go.mongodb.org/mongo-driver/bson"
)

type useGameConfig struct {
	Signature  solana.Signature `json:"signature"`
	ClientSeed []byte           `json:"clientSeed" bson:"clientSeed"`

	TokenTicker string           `json:"tokenTicker" bson:"tokenTicker"`
	Difficulty  tower.Difficulty `json:"difficulty" bson:"difficulty"`
}

func HandleCreateTowerPOST(c *fiber.Ctx) error {
	body := new(useGameConfig)

	if err := c.BodyParser(body); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	var token database.Token
	err := database.FindOne("tokens", bson.M{"ticker": body.TokenTicker}, &token)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid token"))
	}

	gameConfig := tower.GameConfig{
		Signature:  body.Signature,
		ClientSeed: body.ClientSeed,

		Token:      token,
		Difficulty: body.Difficulty,
	}

	game, err := tower.NewGame(gameConfig)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(game)
}

type useActionType struct {
	Level int `json:"level" bson:"level"`
	Tile  int `json:"tile" bson:"tile"`
}

func HandleActionTowerPOST(c *fiber.Ctx) error {
	body := new(useActionType)

	if err := c.BodyParser(body); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	id := c.Params("+")
	if id == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("id cannot be empty"))
	}

	action := tower.ActionType{
		GameID: id,

		Level: body.Level,
		Tile:  body.Tile,
	}

	game, err := tower.Action(action)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(game)
}

func HandleCashoutTowerPOST(c *fiber.Ctx) error {
	id := c.Params("+")
	if id == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("id cannot be empty"))
	}

	cashout := tower.CashoutType{
		GameID: id,
	}

	game, err := tower.Cashout(cashout)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(game)
}

func HandleGetTowerGET(c *fiber.Ctx) error {
	id := c.Params("+")
	if id == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("id cannot be empty"))
	}

	game, err := tower.Get(id)
	if err != nil {
		return JSONError(c, fiber.StatusNotFound, err)
	}

	return c.Status(fiber.StatusOK).JSON(game)
}

func HandleGetTowerRefundGET(c *fiber.Ctx) error {
	id := c.Params("+")
	if id == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("id cannot be empty"))
	}

	signature, _ := solana.SignatureFromBase58(id)
	filter := bson.M{
		"signature": bson.M{
			"$eq": signature,
		},
	}

	var tower tower.Game
	// Execute the find operation and collect the results
	err := database.FindOne("towers", filter, &tower)
	if err == nil {
		// Found the tower
		return c.Status(fiber.StatusOK).JSON(tower)
	}

	var refund database.Refund
	// Execute the find operation and collect the results
	err = database.FindOne("refund", filter, &refund)
	if err == nil {
		// Found the refund
		return c.Status(fiber.StatusOK).JSON(refund)
	}

	return c.Status(fiber.StatusOK).JSON("No data found. Can be refunded.")
}
