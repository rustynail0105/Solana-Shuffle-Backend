package tower

import (
	"context"
	"errors"
	"math"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

func Cashout(cashout CashoutType) (*Game, error) {
	lockGameID(cashout.GameID)
	defer unlockGameID(cashout.GameID)
	game, err := getGame(cashout.GameID)
	if err != nil {
		return nil, err
	}
	if !game.Active {
		return nil, errors.New("game not active")
	}

	game.CalculateMultipliers()

	amount := int(float64(game.BetAmount) * game.Multiplier)

	if amount > env.TowerMaxPayout() {
		amount = env.TowerMaxPayout()
	}

	var instructions []solana.Instruction
	if game.Token.PublicKey == solana.SolMint {
		instructions = vsolana.SendSOLInstructions(
			env.TowerHouse(),
			game.PublicKey,
			amount,
		)
	} else {
		instructions, err = vsolana.CreateAccountAndSendTokenInstructions(
			env.TowerHouse(),
			game.PublicKey,
			game.Token.PublicKey,
			amount,
		)
		if err != nil {
			return nil, err
		}
	}

	signature, err := vsolana.EnsureInstructions([]solana.PrivateKey{env.TowerHouse()}, env.TowerHouse(), instructions)
	if err != nil {
		return nil, err
	}

	cashoutResult := CashoutResult{
		Signature: signature,
		Amount:    amount,
		Done:      true,
	}

	game.CashoutResult = cashoutResult
	game.Active = false

	database.MDB.Collection("towers").ReplaceOne(
		context.TODO(),
		bson.M{"id": cashout.GameID},
		game,
	)

	return game, nil
}

func (g *Game) CalculateMultipliers() (float64, float64) {
	if g.Bust {
		g.Multiplier = 0
		g.NextMultiplier = -1

		return 0, -1
	}

	bombsPerRow := g.Difficulty.getBombsPerRow()
	size := g.Difficulty.getSize()
	width := size[0]
	height := size[1]

	winChancePerRow := float64(width-bombsPerRow) / float64(width)

	totalChance := math.Pow(winChancePerRow, float64(g.Tower.Level))

	multiplier := 1 / totalChance
	nextMultiplier := 1 / (totalChance * winChancePerRow)

	if g.Tower.Level == height {
		nextMultiplier = -1
	}

	// Apply house edge, and round to 2 decimal places
	multiplier = math.Floor(multiplier * float64(10_000-env.TowerHouseEdgeBasisPoints()) / 100) / 100
	nextMultiplier = math.Floor(nextMultiplier * float64(10_000-env.TowerHouseEdgeBasisPoints()) / 100) / 100

	if g.Tower.Level == 0 {
		multiplier = 1
	}

	g.Multiplier = multiplier
	g.NextMultiplier = nextMultiplier

	return multiplier, nextMultiplier
}
