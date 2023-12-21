package user

import (
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/shareed2k/goth_fiber"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

func HandlePublicKeyTwitterAuthGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.FormValue("publicKey"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	signature, err := solana.SignatureFromBase58(c.FormValue("signature"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	originalURL := c.FormValue("originalURL")

	msg := []byte(fmt.Sprintf("solanashuffle twitter %s", publicKey.String()))

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

	authURL, err := goth_fiber.GetAuthURL(c)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	return c.Redirect(authURL)
}

func HandleTwitterCallbackGET(c *fiber.Ctx) error {
	twitterUser, err := goth_fiber.CompleteUserAuth(c)
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
		bson.M{"$set": bson.M{"name": twitterUser.Name, "image": twitterUser.AvatarURL}},
		true,
	)

	originalURL := fmt.Sprint(session.Get("originalURL"))
	if originalURL == "" {
		originalURL = "https://solanashuffle.com/"
	}

	return c.Redirect(originalURL)
}

/*var (
	oauth1Config = &oauth1.Config{
		ConsumerKey:    "",
		ConsumerSecret: "",
		CallbackURL:    "https://api.solanashuffle.com/api/user/twitter/callback",
		Endpoint:       twitterOAuth1.AuthorizeEndpoint,
	}
)


func HandlePublicKeyTwitterAuthGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.FormValue("publicKey"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	signature, err := solana.SignatureFromBase58(c.FormValue("signature"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	msg := []byte(fmt.Sprintf("solanashuffle twitter %s", publicKey.String()))

	if !vsolana.VerifySignature(signature, publicKey, msg) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	session, err := database.CookieStore.Get(c)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	session.Set("publicKey", publicKey.String())
	session.Save()

	return c.Redirect(twitter.LoginHandler(oauth1Config, nil))
}

func HandleTwitterRedirectGET(c *fiber.Ctx) error {
	twitter.LoginHandler(oauth1Config, nil)
	return c.Redirect(twitter.LoginHandler(oauth1Config, nil))
}

func HandleTwitterCallbackGET(c *fiber.Ctx) error {
	ctx := c.Context()
	twitterUser, err := twitter.UserFromContext(ctx)
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

	formattedTwitter := &database.Twitter{
		ID:     twitterUser.IDStr,
		Name:   twitterUser.Name,
		Avatar: twitterUser.ProfileImageURLHttps,
	}

	database.UpdateOne(
		"users",
		bson.M{"publicKey": publicKey},
		bson.M{"$set": bson.M{"twitter": formattedTwitter}},
		true,
	)

	return c.Redirect("https://solanashuffle.com/")
}
*/
