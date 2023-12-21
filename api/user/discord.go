package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/ravener/discord-oauth2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/oauth2"
)

var (
	conf = &oauth2.Config{
		Endpoint:     discord.Endpoint,
		Scopes:       []string{discord.ScopeIdentify},
		RedirectURL:  "https://api.solanashuffle.com/api/user/discord/callback",
		ClientID:     "1048488681511591966",
		ClientSecret: "AcSc9L3sVEShsCymXt46qt41UacwL3Mc",
	}

	state = "moon"
)

func HandlePublicKeyAuthGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.FormValue("publicKey"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	signature, err := solana.SignatureFromBase58(c.FormValue("signature"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	originalURL := c.FormValue("originalURL")

	msg := []byte(fmt.Sprintf("solanashuffle discord %s", publicKey.String()))

	if !vsolana.VerifySignature(signature, publicKey, msg) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	session, err := database.CookieStore.Get(c)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	session.Set("publicKey", publicKey.String())
	session.Set("originalURL", originalURL)
	session.Save()

	authURL := conf.AuthCodeURL(state)

	return c.Redirect(authURL)
}

func HandleDiscordCallbackGET(c *fiber.Ctx) error {
	if c.FormValue("state") != state {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid state"))
	}

	token, err := conf.Exchange(context.Background(), c.FormValue("code"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	resp, err := conf.Client(context.Background(), token).Get("https://discord.com/api/users/@me")
	if err != nil || resp.StatusCode != 200 {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	var discord database.Discord
	err = json.Unmarshal(body, &discord)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	session, err := database.CookieStore.Get(c)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	publicKey, err := solana.PublicKeyFromBase58(fmt.Sprint(session.Get("publicKey")))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"name": discord.Username, "image": discord.AvatarURL()}},
		true,
	)

	originalURL := fmt.Sprint(session.Get("originalURL"))
	if originalURL == "" {
		originalURL = "https://solanashuffle.com/"
	}

	return c.Redirect(originalURL)
}
