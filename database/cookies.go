package database

import (
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/redis"
)

var (
	CookieStore *session.Store
)

func init() {
	storage := redis.New(redis.Config{
		Host:     "52.70.17.114",
		Port:     6379,
		Password: "kHzJSq5F9WfEVHXI+YT4N5E4nDC9WXje1aU/X0UForBL5MFdSHhudkmZr4sK00aldSS9Fxkj5/gY3g5S2kHPrg==",
		Database: 1,
		Reset:    false,
	})

	CookieStore = session.New(session.Config{
		Storage: storage,
	})
}
