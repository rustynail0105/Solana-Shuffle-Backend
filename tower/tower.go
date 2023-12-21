package tower

import (
	"context"
	"errors"

	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/tower/fair"
	"go.mongodb.org/mongo-driver/bson"
)

func Get(id string) (*Game, error) {
	lockGameID(id)
	defer unlockGameID(id)
	game, err := getGame(id)
	if err != nil {
		return nil, err
	}

	game.Tower.calculatePublicPath()
	game.CalculateMultipliers()
	return game, nil
}

func getGame(id string) (*Game, error) {
	game := new(Game)

	err := database.MDB.Collection("towers").FindOne(
		context.TODO(),
		bson.M{"id": id},
	).Decode(game)
	if err != nil {
		return nil, err
	}

	return game, nil
}

func NewTower(difficulty Difficulty) (*Tower, error) {
	path := generatePath(difficulty)

	tower := Tower{
		InternalPath: path,
		Level:        0,
		Difficulty:   difficulty,
	}

	tower.calculatePublicPath()

	return &tower, nil
}

func generatePath(difficulty Difficulty) path {
	bombsPerRow := difficulty.getBombsPerRow()
	size := difficulty.getSize()
	width := size[0]

	path := emptyPath(size, safe)

	for rowIndex := range path {
		bombIndexes := fair.RandomUniqueIntArray(bombsPerRow, 0, width-1)
		for _, bombIndex := range bombIndexes {
			path[rowIndex][bombIndex] = 1
		}
	}

	return path
}

func emptyPath(size size, value int) path {
	path := make(path, size[1])
	for i := range path {
		path[i] = make([]int, size[0])
		for i2 := range path[i] {
			path[i][i2] = value
		}
	}

	return path
}

func (d Difficulty) getBombsPerRow() int {
	var bombsPerRow int
	switch d {
	case easy, medium, hard:
		bombsPerRow = 1
	case expert:
		bombsPerRow = 2
	case master:
		bombsPerRow = 3
	}

	return bombsPerRow
}

func (d Difficulty) getSize() [2]int {
	return sizes[int(d)]
}

func (d Difficulty) isValid() bool {
	return d == easy || d == medium || d == hard || d == expert || d == master
}

func (t *Tower) calculatePublicPath() path {
	size := t.Difficulty.getSize()

	path := emptyPath(size, unknown)
	for i := 0; i < t.Level; i++ {
		path[i] = t.InternalPath[i]
	}

	t.Path = path
	return path
}

func (c GameConfig) validate() error {
	if c.Signature.IsZero() {
		return errors.New("signature cannot be zero")
	}

	if len(c.ClientSeed) == 0 {
		return errors.New("client seed cannot be empty")
	}

	if !c.Difficulty.isValid() {
		return errors.New("invalid difficulty")
	}

	return nil
}
