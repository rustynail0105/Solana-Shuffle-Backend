package shuffle

import (
	"errors"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/imagelibrary"
	"github.com/solanashuffle/backend/stream"
	"go.mongodb.org/mongo-driver/bson"
)

type CreateRoomConfig struct {
	Creator               solana.PublicKey
	CreatorFeeBasisPoints int
	Name                  string
	Public                bool
	Token                 database.Token

	MinimumAmount int
	MaximumAmount int
	MaxCountdown  time.Duration

	Official bool
}

func NewRoom(config CreateRoomConfig) (*Room, error) {
	if len(config.Name) > 20 {
		return &Room{}, errors.New("name too long")
	}
	if len(config.Name) < 3 {
		return &Room{}, errors.New("name too short")
	}

	r := Room{
		ID: uuid.NewString(),

		CreationTime: time.Now().Unix(),
		CoverImage:   imagelibrary.Random(),

		Creator:               config.Creator,
		CreatorFeeBasisPoints: config.CreatorFeeBasisPoints,
		Name:                  config.Name,
		Public:                config.Public,

		Official: config.Official,

		Token: config.Token,

		Config: RoomConfig{
			MinimumAmount: config.MinimumAmount,
			MaximumAmount: config.MaximumAmount,
		},

		Stats: database.Stats{
			TotalVolume: 0,
			TotalGames:  0,
			Volumes:     make(map[string]uint64),
			Games:       make(map[string]uint64),
		},
	}

	r.Official = config.Creator == env.House().PublicKey()

	err := database.FindOne(
		"rooms",
		bson.M{"name": config.Name},
		nil,
	)
	if err == nil {
		return &Room{}, errors.New("name is already used")
	}

	err = database.FindOne(
		"rooms",
		bson.M{"creator": config.Creator},
		nil,
	)
	if err == nil {
		return &Room{}, errors.New("you can only create one room")
	}

	database.InsertOne(
		"rooms",
		r,
	)

	return GetRoom(r.ID)
}

func (r *Room) Init() {
	r.stream = stream.New()
	go r.stream.Start()
	r.NewSession()
	go r.Routine()
}

func (rc RoomConfig) CheckBetAmount(amount int) error {
	if rc.MinimumAmount > 0 {
		if amount < rc.MinimumAmount {
			return errors.New("bet too low")
		}
	}
	if rc.MaximumAmount > 0 {
		if amount > rc.MaximumAmount {
			return errors.New("bet too high")
		}
	}

	return nil
}
