package shuffle

import (
	"errors"
	"sync"

	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	rooms     = make(map[string]*Room)
	roomsLock = sync.RWMutex{}
)

func InitRooms() {
	var rooms []Room
	err := database.Find(
		"rooms",
		bson.M{
			"id": "2c15efe5-6187-4bfe-980e-e25a91a1a762",
		},
		&rooms,
	)
	if err != nil {
		return
	}

	for _, r := range rooms {
		GetRoom(r.ID)
	}
}

func GetRoom(id string) (*Room, error) {
	roomsLock.RLock()

	r, ok := rooms[id]
	roomsLock.RUnlock()
	if !ok {
		var r Room
		err := database.FindOne(
			"rooms",
			bson.M{"id": id},
			&r,
		)
		if err != nil {
			return &Room{}, errors.New("not found")
		}

		r.Init()
		roomsLock.Lock()
		rooms[id] = &r
		roomsLock.Unlock()
		return GetRoom(id)
	}

	return r, nil
}
