package tower

import (
	"context"
	"errors"

	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"
)

func Action(action ActionType) (*Game, error) {
	lockGameID(action.GameID)
	defer unlockGameID(action.GameID)
	game, err := getGame(action.GameID)
	if err != nil {
		return nil, err
	}

	if !game.Active {
		return nil, errors.New("game not active")
	}

	size := game.Difficulty.getSize()
	width := size[0]
	height := size[1]

	if action.Level != game.Tower.Level+1 {
		return nil, errors.New("incorrect level")
	}
	if action.Level > height {
		return nil, errors.New("max win reached, need to cash out")
	}
	if action.Tile > width-1 || action.Tile < 0 {
		return nil, errors.New("invalid tile")
	}

	row := game.Tower.InternalPath[action.Level-1]
	tile := row[action.Tile]

	defer database.MDB.Collection("towers").ReplaceOne(
		context.TODO(),
		bson.M{"id": action.GameID},
		game,
	)

	game.Tower.Level += 1

	if tile == bomb {
		game.Tower.InternalPath[action.Level-1][action.Tile] = clickedBomb

		game.Bust = true
		game.Active = false
		game.CalculateMultipliers()
		game.Tower.calculatePublicPath()
		return game, nil
	}

	game.Tower.InternalPath[action.Level-1][action.Tile] = clickedSafe
	game.CalculateMultipliers()
	game.Tower.calculatePublicPath()

	return game, nil
}
