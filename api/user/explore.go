package user

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/utility"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	roomsCache               []shuffle.Room
	roomsCacheByTotalVolume  []shuffle.Room
	roomsCacheByTodayVolume  []shuffle.Room
	roomsCacheByCreationTime []shuffle.Room

	cacheLock = &sync.Mutex{}
)

func ExploreCache() {
	go func() {
		for {
			err := database.Find("rooms", bson.M{"public": true}, &roomsCache)
			if err != nil {
				continue
			}
			cacheLock.Lock()
			roomsCacheByTotalVolume = make([]shuffle.Room, len(roomsCache))
			roomsCacheByTodayVolume = make([]shuffle.Room, len(roomsCache))
			roomsCacheByCreationTime = make([]shuffle.Room, len(roomsCache))
			copy(roomsCacheByTotalVolume, roomsCache)
			copy(roomsCacheByTodayVolume, roomsCache)
			copy(roomsCacheByCreationTime, roomsCache)
			sort.Slice(roomsCacheByTotalVolume, func(i, j int) bool {
				return roomsCacheByTotalVolume[i].Stats.TotalVolume > roomsCacheByTotalVolume[j].Stats.TotalVolume
			})

			for i, room := range roomsCacheByTodayVolume {
				if room.Stats.Volumes == nil {
					room.Stats.Volumes = make(map[string]uint64)
				}
				if _, ok := room.Stats.Volumes[utility.FormatDate(time.Now())]; !ok {
					room.Stats.Volumes[utility.FormatDate(time.Now())] = 0
					roomsCacheByTodayVolume[i] = room
				}

				if room.Stats.Games == nil {
					room.Stats.Games = make(map[string]uint64)
				}
				if _, ok := room.Stats.Games[utility.FormatDate(time.Now())]; !ok {
					room.Stats.Games[utility.FormatDate(time.Now())] = 0
					roomsCacheByTodayVolume[i] = room
				}
			}

			sort.Slice(roomsCacheByTodayVolume, func(i, j int) bool {
				return roomsCacheByTodayVolume[i].Stats.Volumes[utility.FormatDate(time.Now())] > roomsCacheByTodayVolume[j].Stats.Volumes[utility.FormatDate(time.Now())]
			})

			sort.Slice(roomsCacheByCreationTime, func(i, j int) bool {
				return roomsCacheByCreationTime[i].CreationTime > roomsCacheByCreationTime[j].CreationTime
			})
			cacheLock.Unlock()

			time.Sleep(time.Second * 10)
		}
	}()
}

func HandleCreatorRoomsGET(c *fiber.Ctx) error {
	creator, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	signature, err := solana.SignatureFromBase58(c.FormValue("signature"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	wanted := fmt.Sprintf("solanashuffle my rooms %s", creator.String())
	if !vsolana.VerifySignature(signature, creator, []byte(wanted)) {
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid signature"))
	}

	rooms := []shuffle.Room{}
	database.Find("rooms", bson.M{"creator": creator}, &rooms)
	return c.Status(fiber.StatusOK).JSON(rooms)
}

func HandleFavoriteRoomsGET(c *fiber.Ctx) error {
	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	user := database.NewUser(publicKey)
	err = database.FindOne("users", bson.M{"publicKey": publicKey}, &user)
	if err != nil {
		return JSONError(c, fiber.StatusNotFound, errors.New("you have no account yet"))
	}

	if len(user.FavoriteRooms) == 0 {
		return c.Status(fiber.StatusOK).JSON([]shuffle.Room{})
	}

	var rooms []shuffle.Room
	filters := []bson.M{}
	for _, id := range user.FavoriteRooms {
		filters = append(filters, bson.M{"id": id})
	}
	err = database.Find("rooms", bson.M{"$or": filters}, &rooms)
	if err != nil {
		return JSONError(c, fiber.StatusNotFound, errors.New("rooms not found"))
	}

	return c.Status(fiber.StatusOK).JSON(rooms)
}

func HandleExploreRoomsGET(c *fiber.Ctx) error {
	//query := c.FormValue("q")
	cacheLock.Lock()
	defer cacheLock.Unlock()
	sort := c.FormValue("sort")
	var sortedRooms []shuffle.Room
	switch sort {
	case "totalVolume":
		sortedRooms = roomsCacheByTotalVolume
	case "todayVolume":
		sortedRooms = roomsCacheByTodayVolume
	case "creationTime":
		sortedRooms = roomsCacheByCreationTime
	default:
		return JSONError(c, fiber.StatusBadRequest, errors.New("invalid sort"))
	}

	exploreRooms := make([]shuffle.Room, len(sortedRooms))
	copy(exploreRooms, sortedRooms)

	maxi := 20
	if maxi > len(exploreRooms) {
		maxi = len(exploreRooms)
	}

	return c.Status(fiber.StatusOK).JSON(exploreRooms[:maxi])
}
