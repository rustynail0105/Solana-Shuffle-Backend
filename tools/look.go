package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/joho/godotenv"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/tower"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	env.Set("mainnet-beta")
	database.ConnectDatabases()
}

func main() {
	checkShuffle()
}

func prettyPrint(i interface{}) string {
    s, _ := json.MarshalIndent(i, "", "\t")
    return string(s)
}

func checkShuffle() {
	signature, _ := solana.SignatureFromBase58("5ZVvsqvD6dyKzV2j3NjJJGrv4wvtoMwCrS8wFRVfYSE8YiCZG133r3GMFVwWmcMP55hwF76576V4QXfWkKvN2zLe")

	filter := bson.M{"users.signatures": signature}

	var session shuffle.Session
	// Execute the find operation and collect the results
	err := database.FindOne("sessions", filter, &session)

	if err == nil {
		fmt.Println(prettyPrint(session))
		var value float64
		for _, user := range session.Users {
			value += float64(user.Assets.Value()) / float64(1_000_000_000)
		}
		fmt.Println("Total value: ", value)
	} else {
		fmt.Println(err)
	}

	// Check refund signatures
	filter = bson.M{"refundSignatures": signature}

	err = database.FindOne("sessions", filter, &session)

	if err == nil {
		fmt.Println(prettyPrint(session))
	} else {
		fmt.Println(err)
	}

	// Check result signatures
	filter = bson.M{"result.signatures": signature}

	err = database.FindOne("sessions", filter, &session)

	if err == nil {
		fmt.Println(prettyPrint(session))
	} else {
		fmt.Println(err)
	}
}

func checkTowerRefund() {
	fmt.Println("checking game history...")
	id := "2tJthkhcRuSryGg85miQtDCoPfS96EA73dgDQZCu6hbkF3XpRKvNa4YroEj8ubdV9AdynBWfUvxyVtk9WTNptNcY"
	refundTX := true
	
	if id == "" {
		fmt.Println("id cannot be empty")
		return
	}

	signature, _ := solana.SignatureFromBase58(id)

	filter := bson.M{
		"signature": bson.M{
			"$eq": signature,
		},
	}

	var towerGame tower.Game
	// Execute the find operation and collect the results
	err := database.FindOne("towers", filter, &towerGame)
	if err == nil {
		// Found the tower
		fmt.Println(prettyPrint(towerGame))
		if refundTX {
			cashout := tower.CashoutType{
				GameID: towerGame.ID,
			}
			_, err := tower.Cashout(cashout)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
		return
	}

	var refund database.Refund
	// Execute the find operation and collect the results
	err = database.FindOne("refund", filter, &refund)
	if err == nil {
		// Found the refund
		fmt.Println(refund)
		return
	}

	fmt.Println("No data found. Can be refunded.")
}

func checkTower() {
	fmt.Println("starting...")
	publicKey, _ := solana.PublicKeyFromBase58("FpBkZUicffgLhma5g272hZ8zAnb3pPjvSnYPw118rEEc")
	// signature, _ := solana.SignatureFromBase58("1111111111111111111111111111111111111111111111111111111111111111")
	filter := bson.M{
		// Check for refunds within the last 10 hours
		"creationTime": bson.M{
			"$gt": time.Now().Add(-10 * time.Minute).Unix(),
		},
		"publicKey": bson.M{
			"$eq": publicKey,
		},
		// "signature": bson.M{
		// 	"$eq": signature,
		// },
	}

	var towers []tower.Game
	// Execute the find operation and collect the results
	err := database.Find("towers", filter, &towers)
	if err != nil {
		log.Println(err)
		return
	}

	// print the results
	for _, tower := range towers {
		fmt.Println(prettyPrint(tower))
	}
}

func checkRefund() {
	fmt.Println("starting...")
	filter := bson.M{
		// Check for refunds within the last 24 hours
		"creationTime": bson.M{
			"$gt": time.Now().Add(-24 * time.Hour).Unix(),
		},
		"game": bson.M{
			"$eq": "shuffle",
		},
	}

	var refunds []database.Refund
	// Execute the find operation and collect the results
	err := database.Find("refund", filter, &refunds)
	if err != nil {
		log.Println(err)
		return
	}

	// print the results
	for _, refund := range refunds {
		fmt.Println(refund)
	}
}

func exportCSV() {
	fmt.Println("starting...")
	var users []database.UserProfile
	err := database.Find("users", bson.M{}, &users)
	if err != nil {
		panic(err)
	}

	outMap := make(map[solana.PublicKey]int)

	for _, u := range users {
		if vol, ok := u.Stats.Volumes["2023-02-14"]; ok && vol/1_000_000_000 > 0 {
			outMap[u.PublicKey] = int(vol) / 1_000_000_000
		}
	}

	data := [][]string{}

	for k, v := range outMap {
		data = append(data, []string{k.String(), strconv.Itoa(v)})

	}

	csvFile, err := os.Create("data.csv")
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvFile)

	for _, empRow := range data {
		err = csvwriter.Write(empRow)
		if err != nil {
			panic(err)
		}
	}

	csvwriter.Flush()
	csvFile.Close()

	/*


		log.Println("got sessions")

		fmt.Println(len(sessions))

		for _, session := range sessions {
			fmt.Println(
			for _, priv := range session.IntermediaryAccounts {
				if priv.PublicKey() == solana.MustPublicKeyFromBase58("8bvcd78FWuS3bPE8EQjcs91V7WM2NxVX7SfaatY2jn4S") {
					fmt.Println(priv)

					fmt.Println(session.IntermediaryAccounts)
					fmt.Println(session.Assets().Value())
					fmt.Println(session.Result.Winner)
					fmt.Println(session.RoomID)
					session.WaitUntilAssetsFinalized2(database.Token{
						PublicKey: solana.SolMint,
						Decimals:  9,
						Ticker:    "SOL",
					})

					for _, user := range session.Users {
						if user.PublicKey == solana.MustPublicKeyFromBase58("39FniKsSeSyv7TuqjK7LqQaQiC7v7YMwJeE1sSDyU7EE") {
							fmt.Println("found")
						}
					}

					fmt.Println(session.Assets().Value())
					fmt.Println("done waiting")

					//signatures, err := session.SendWinnings(solana.MustPublicKeyFromBase58("AJ5rsGhTKaNPGGJbWXZtXT42PHF75kaF2V42b5n2pino"))
					//fmt.Println(err)
					//fmt.Println(signatures)

				}
			}
		}
	*/
}

/*
func main() {
	account := solana.MustPublicKeyFromBase58("ChrigMdavfg46LdVD8TEUx29rTShd5VxYgtJ1CB4XUAP")

	token := database.Token{
		// PublicKey: solana.MustPublicKeyFromBase58("DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263"),
		PublicKey: solana.SolMint,
	}

	u := new(shuffle.SessionUser)

	u.PublicKey = account
	u.Assets = shuffle.Assets{
		{
			Type:  "Token",
			Price: 133019280,
			Mint:  solana.SolMint,
		},
	}

	log.Println("looking for deposit")
	log.Println("current balance: 133019280")

	err := u.ParseIntermediary(token, 1000, "Token")
	if err != nil {
		panic(err)
	}
	log.Println("found higher balance")
	log.Println(u)
}
*/
