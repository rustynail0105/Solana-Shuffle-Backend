package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/api/user"
)

func SetApiGroup(group fiber.Router) {
	userGroup := group.Group("/user")
	user.SetUserGroup(userGroup)
}
