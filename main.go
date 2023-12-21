package main

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"github.com/solanashuffle/backend/api"
	"github.com/solanashuffle/backend/api/user"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/shuffle/conversion"
	"github.com/solanashuffle/backend/shuffle/leaderboards"
	"github.com/solanashuffle/backend/shuffle/refund"
	"github.com/solanashuffle/backend/vsolana"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	env.Set("mainnet-beta")
	vsolana.ConnectWS(env.GetWSURL())
	database.ConnectDatabases()
	shuffle.InitRooms()
	user.ExploreCache()
	leaderboards.Routine()
	conversion.Routine()
	refund.Routine()
}

func main() {
	app := fiber.New(fiber.Config{
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		IdleTimeout:  time.Second * 5,
	})
	app.Use(cors.New())

	apiGroup := app.Group("/api")
	api.SetApiGroup(apiGroup)

	app.Listen(env.GetPort())
}
