package user

import (
	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/twitter"
	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"
)

func SetUserGroup(group fiber.Router) {
	goth.UseProviders(
		twitter.New("T3pNTFEycXI1WE40V3FES1QteTE6MTpjaQ", "poOFHnPQC_RCJi-Z0ft-hiOTHfnu99D9-ywJTLFHYIM1teSXz3", "https://api.solanashuffle.com/api/user/twitter/callback"),
	)

	// << Tower Stuff >> //
	group.Post("/tower/+/action", HandleActionTowerPOST)
	group.Post("/tower/+/cashout", HandleCashoutTowerPOST)
	group.Get("/tower/+", HandleGetTowerGET)
	group.Get("/tower/refund/+", HandleGetTowerRefundGET)
	group.Post("/tower", HandleCreateTowerPOST)
	// << Tower Stuff >> //

	// << Profile Stuff >> //
	// ProfileNameSET
	group.Post("/user/name/change/+", HandleProfileNameSET)
	// ProfileImageSet
	group.Post("/user/image/change/+", HandleProfileImageSET)
	// BannerSet
	group.Post("/user/banner/change/+", HandleProfileBannerSET)
	// ProfileGet
	group.Get("/user/profile/+", HandleProfileGET)
	// ProfileHistoryGET
	group.Get("/user/history/+", HandleProfileHistoryGET)
	// << Profile Stuff >> //

	// << Level Handling >> //
	// << Level Handling >> //
	group.Get("/user/+", HandleUserGET)

	group.Get("/leaderboards", HandleLeaderboardsGET)

	group.Get("/nft/+", HandleNFTGET)
	//group.Get("/nfts/+", HandleNFTBatchGet)
	group.Get("/nfts/+", HandleUserAssetsGET)

	group.Get("/tokens", HandleTokensGET)
	group.Get("/rooms/favorites/+", HandleFavoriteRoomsGET)
	group.Get("/rooms/active-count/", HandleRoomUserCountGET)
	group.Get("/rooms/explore", HandleExploreRoomsGET)
	group.Get("/rooms/creator/+", HandleCreatorRoomsGET)
	group.Post("/room/+/join", HandleJoinRoomPOST)
	//group.Post("/room/+/intermediary", HandleInitJoinRoomPOST)
	group.Get("/room/+/opened", HandleRoomOpenedGET)
	group.Post("/room/+/favorite", HandleFavoriteRoomPOST)
	group.Post("/room/+/unfavorite", HandleUnfavoriteRoomDELETE)
	group.Get("/room/+/history", HandleRoomHistoryGET)
	group.Get("/room/+", HandleGetRoomGET)
	group.Post("/room", HandleCreateRoomPOST)

	group.Get("/discord/redirect", HandlePublicKeyAuthGET)
	group.Get("/discord/callback", HandleDiscordCallbackGET)

	// twitter completely broken, removed for now
	//group.Get("/twitter/redirect", HandlePublicKeyTwitterAuthGET)
	//group.Get("/twitter/callback", HandleTwitterCallbackGET)

	wsGroup := group.Group("/ws")
	SetWSGroup(wsGroup)

}

func HandleUserGET(c *fiber.Ctx) error {
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
		database.InsertOne("users", user)
	}

	return c.Status(fiber.StatusOK).JSON(user)
}

func JSONError(c *fiber.Ctx, status int, err error) error {
	return c.Status(status).JSON(fiber.Map{
		"message": err.Error(),
	})
}
