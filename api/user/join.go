package user

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/shuffle"
)

type joinBody struct {
	Signature solana.Signature `json:"signature"`
}

/*
	func HandleInitJoinRoomPOST(c *fiber.Ctx) error {
		body := new(joinBody)
		err := c.BodyParser(body)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, err)
		}

		if body.PublicKey.IsZero() {
			return JSONError(c, fiber.StatusBadRequest, errors.New("invalid publickey"))
		}

		roomId := c.Params("+")
		room, err := shuffle.GetRoom(roomId)
		if err != nil {
			return JSONError(c, fiber.StatusNotFound, err)
		}

		intermediary, err := room.InitJoin2(body.PublicKey)
		if err != nil {
			return JSONError(c, fiber.StatusBadRequest, err)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"intermediary": intermediary,
		})
	}
*/
func HandleJoinRoomPOST(c *fiber.Ctx) error {
	body := new(joinBody)
	err := c.BodyParser(body)
	if err != nil {
		fmt.Println(err)
		return JSONError(c, fiber.StatusBadRequest, err)
	}
	if body.Signature.IsZero() {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid publicKey"))
	}

	roomId := c.Params("+")
	room, err := shuffle.GetRoom(roomId)
	if err != nil {
		return JSONError(c, fiber.StatusNotFound, err)
	}

	sessionUser, err := room.Join(body.Signature)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(sessionUser)
}

func HandleRoomOpenedGET(c *fiber.Ctx) error {
	roomId := c.Params("+")
	room, err := shuffle.GetRoom(roomId)
	if err != nil {
		return JSONError(c, fiber.StatusNotFound, err)
	}

	return c.Status(fiber.StatusOK).JSON(room.Session.IsPubliclyOpen())
}
