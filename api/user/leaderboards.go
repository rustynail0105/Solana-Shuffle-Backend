package user

import (
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/shuffle/leaderboards"
)

func HandleLeaderboardsGET(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"totalVolume": leaderboards.TotalVolumeUsers(),
		"todayVolume": leaderboards.TodayVolumeUsers(),
	})
}
