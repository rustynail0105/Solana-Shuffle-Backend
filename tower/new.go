package tower

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/google/uuid"
	"github.com/solanashuffle/backend/vsolana"
)

var (
	graceAmount = 500_000
)

func NewGame(config GameConfig) (*Game, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	log.Println("awaiting confirmation")
	err := vsolana.AwaitConfirmedTransaction(config.Signature)
	if err != nil {
		log.Println(err)
		var refund database.Refund
		err = database.FindOne("refund", bson.M{"signature": config.Signature}, &refund)
		// Only refund if the transaction has not been refunded before
		if err != nil {
			database.InsertOne("refund", database.Refund{
				Signature:    config.Signature,
				Token:        config.Token,
				CreationTime: time.Now().Unix(),
				RefundStatus: "pending",
				Game:         "tower",
			})
		}
		return &Game{}, errors.New("transaction failed")
	}

	log.Println("parsing tx")
	parsedTransaction, err := ParseTransaction(config.Signature, config.Token, true, true)
	if err != nil {
		return &Game{}, err
	}

	//check fee amount
	desiredFeeAmount := parsedTransaction.BetAmount*env.TowerFeeBasisPoints()/10_000 - graceAmount
	if parsedTransaction.FeeAmount < desiredFeeAmount {
		return &Game{}, errors.New("fee not paid")
	}

	log.Println("creating tower")
	tower, err := NewTower(config.Difficulty)
	if err != nil {
		return &Game{}, err
	}

	g := Game{
		ID: uuid.NewString(),

		Active: true,

		Tower:      *tower,
		Multiplier: 1,
		Difficulty: config.Difficulty,

		PublicKey: parsedTransaction.PublicKey,
		Signature: config.Signature,

		Token:        config.Token,
		BetAmount:    parsedTransaction.BetAmount,
		FeeAmount:    parsedTransaction.FeeAmount,
		CreationTime: time.Now().Unix(),
	}

	_, err = database.MDB.Collection("towers").InsertOne(
		context.TODO(),
		g,
	)
	if err != nil {
		return &Game{}, err
	}

	return &g, nil
}
