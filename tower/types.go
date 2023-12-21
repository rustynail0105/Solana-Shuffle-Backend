package tower

import (
	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
)

type Difficulty int

const (
	easy   Difficulty = 0 // 4 columns, 1 bomb per row
	medium Difficulty = 1 // 3 columns, 1 bomb per row
	hard   Difficulty = 2 // 2 columns, 1 bomb per row

	expert Difficulty = 3 // 3 columns, 2 bombs per row
	master Difficulty = 4 // 4 columns, 3 bombs per row

	unknown int = 9

	safe int = 0
	bomb int = 1

	clickedSafe int = 2
	clickedBomb int = 3
)

var sizes = [][2]int{{4, 9}, {3, 9}, {2, 9}, {3, 9}, {4, 9}}

type GameConfig struct {
	Signature  solana.Signature `json:"signature"`
	ClientSeed []byte           `json:"clientSeed" bson:"clientSeed"`

	Token      database.Token      `json:"token" bson:"token"`
	Difficulty Difficulty `json:"difficulty" bson:"difficulty"`
}

type ParsedTransactionData struct {
	Signature solana.Signature
	PublicKey solana.PublicKey

	BetAmount int
	FeeAmount int
}

type Game struct {
	ID string `json:"id" bson:"id"`

	Active bool `json:"active" bson:"active"`

	CashoutResult  CashoutResult `json:"cashoutResult" bson:"cashoutResult"`
	Bust           bool          `json:"bust" bson:"bust"`
	Multiplier     float64       `json:"multiplier" bson:"multiplier"`
	NextMultiplier float64       `json:"nextMultiplier" bson:"nextMultiplier"`

	Tower      Tower      `json:"tower" bson:"tower"`
	Difficulty Difficulty `json:"difficulty" bson:"difficulty"`

	PublicKey solana.PublicKey `json:"publicKey" bson:"publicKey"`
	Signature solana.Signature `json:"signature" bson:"signature"`

	Token        database.Token `json:"token" bson:"token"`
	BetAmount    int   `json:"betAmount" bson:"betAmount"`
	FeeAmount    int   `json:"feeAmount" bson:"feeAmount"`
	CreationTime int64 `json:"creationTime" bson:"creationTime"`
}

type Tower struct {
	InternalPath path       `json:"-" bson:"internalPath"`
	Path         path       `json:"path" bson:"path"`
	Level        int        `json:"level" bson:"level"`
	Difficulty   Difficulty `json:"difficulty" bson:"difficulty"`
}

type ActionType struct {
	GameID string `json:"gameId"`

	Level int `json:"level" bson:"level"`
	Tile  int `json:"tile" bson:"tile"`
}

type CashoutType struct {
	GameID string `json:"gameId"`
}

type CashoutResult struct {
	Signature solana.Signature `json:"signature" bson:"signature"`
	Amount    int              `json:"amount" bson:"amount"`

	Done bool `json:"done" bson:"done"`
}

type path [][]int

type size [2]int