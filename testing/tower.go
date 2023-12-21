package main

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/tower"
)

func main() {
	test_game_multiplier()
}

func test_game_multiplier() {
	sol := database.Token{
		Ticker:    "SOL",
		PublicKey: solana.SolMint,
		Decimals:  9,
	}
	signature, _ := solana.SignatureFromBase58("3hiqK3rJ6skneQAWxrCJ9mxAjVBzcB2UUoftmYegcqMHHdJ6MHHzGmKPCYcDnbhNuLoeVkecc8SZpa8L3LA13d6c")
	parsedTransaction, _ := tower.ParseTransaction(signature, sol, false, false)
	newTower, _ := tower.NewTower(0)

	g := tower.Game{
		ID: uuid.NewString(),

		Active: true,

		Tower:      *newTower,
		Multiplier: 1,
		Difficulty: 0,

		PublicKey: parsedTransaction.PublicKey,

		Token:        sol,
		BetAmount:    parsedTransaction.BetAmount,
		FeeAmount:    parsedTransaction.FeeAmount,
		CreationTime: time.Now().Unix(),
	}

	multiplier, nextMultiplier := g.CalculateMultipliers()
	fmt.Println(multiplier, nextMultiplier)
}