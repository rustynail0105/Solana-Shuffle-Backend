package user

import (
	"errors"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
	"time"
)

type useReferralBody database.Referral

func HandleCreateReferralPost(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(useReferralBody)
	err = c.BodyParser(body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	wanted := []byte(fmt.Sprintf(
		"solanashuffle referral %s %s", body.MyLink, publicKey.String(),
	))

	if !vsolana.VerifySignature(body.Signature, body.PublicKey, wanted) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	if body.MyLink == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("myLink missing"))
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": body},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}
	return c.Status(fiber.StatusOK).JSON(body)
}

func HandleUseReferralPost(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	body := new(useReferralBody)
	err = c.BodyParser(body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	wanted := []byte(fmt.Sprintf(
		"solanashuffle referral %s %s", body.PlayLink, publicKey.String(),
	))

	if !vsolana.VerifySignature(body.Signature, body.PublicKey, wanted) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	if body.PlayLink == "" {
		return JSONError(c, fiber.StatusBadRequest, errors.New("playLink missing"))
	}

	user, err := database.DbGetUser(publicKey)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	if user.Stats.TotalGames > 0 || user.Referral.Used {
		return c.Status(fiber.StatusBadRequest).JSON("not eligible to use referrals")
	}

	referralDoc := database.Referral{
		PlayLink: body.PlayLink,
		Wallet:   body.Wallet,
		Earned:   0,
		Used:     false, // this is set after the 24 hours
		Activated: time.Now(),
	}

	if err := database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": referralDoc},
		false,
	); err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Status(fiber.StatusOK).JSON(body)
}