package refund

import (
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/tower"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

func Routine() {
	go func() {
		routine()
	}()
}

func sendRefundForShuffle(sessionUser shuffle.SessionUser, token database.Token) ([]solana.Signature, error) {
	// Do not refund the fees for now, as this will slowly drain the shuffle account
	// sessionUser.Assets = append(sessionUser.Assets, shuffle.GeneralAsset{
	// 	Type: "Token",
	// 	Price: sessionUser.Assets.Value() * env.FeeBasisPoints() / 10_000,
	// 	Mint:  token.PublicKey,
	// })
	signatures, err := sessionUser.Assets.SendAndAwaitConfirmation(
		env.House(),
		sessionUser.PublicKey,
	)
	if err != nil {
		return nil, error(fmt.Errorf("failed to send refund tx"))
	}
	time.Sleep(time.Second * 5)
	log.Println("Refund TX sent: ", signatures)
	return signatures, nil
}

func sendRefundForTower(parsedTransactionData tower.ParsedTransactionData, token database.Token) ([]solana.Signature, error) {
	if token.PublicKey != solana.SolMint {
		return nil, error(fmt.Errorf("Token is not SOL"))
	}
	instructions := vsolana.SendSOLInstructions(
		env.TowerHouse(),
		parsedTransactionData.PublicKey,
		parsedTransactionData.BetAmount + parsedTransactionData.FeeAmount,
	)
	signature, err := vsolana.EnsureInstructions([]solana.PrivateKey{env.TowerHouse()}, env.TowerHouse(), instructions)
	if err != nil {
		return nil, error(fmt.Errorf("Failed to send refund tx"))
	}
	time.Sleep(time.Second * 5)
	log.Println("Refund TX sent: ", signature)
	return []solana.Signature {signature}, nil
}

// Function to insert a refund into the database for testing
func insertRefundForTesting() error {
	// Create a SOL token
	sol := database.Token{
		Ticker:    "SOL",
		PublicKey: solana.SolMint,
		Decimals:  9,
	}
	signature, _ := solana.SignatureFromBase58("MDqAfCpebQfpzfAeJUipkNWJaPS4ZUjNAi4GXVkyJY43ceWCemKRrZbB8t7Enqk6YyXWkMwJQiPgd3wJYz6XDa6")
	// Insert the refund into the database
	refund := database.Refund{
		Signature:    signature,
		Token:        sol,
		RefundStatus: "pending",
		CreationTime: time.Now().Unix(),
		Game: 		  "tower",
	}
	err := database.InsertOne("refund", refund)
	if err != nil {
		return err
	}
	return nil
}

func routine() {
	for {
		// Only try to refund the pending tx in the last 10 minutes
		filter := bson.M{
			"creationTime": bson.M{
				"$gt": time.Now().Add(-10 * time.Minute).Unix(),
			},
			"refundStatus": "pending",
		}

		var refunds []database.Refund
		// Execute the find operation and collect the results
		err := database.Find("refund", filter, &refunds)
		if err != nil {
			log.Println(err)
			return
		}

		for _, refund := range refunds {
			// Check if the transaction has been confirmed
			err := vsolana.AwaitConfirmedTransaction(refund.Signature)
			if err != nil {
				log.Println(err)
				continue
			}

			// Before refund, make sure the refund is still pending
			err = database.FindOne("refund", filter, &refund)
			if err != nil {
				log.Println(err)
				continue
			}
			// Another instance of the refund routine may have already processed this refund
			if refund.RefundStatus != "pending" {
				continue
			}
			// Mark the refund as processing
			database.UpdateOne(
				"refund",
				bson.M{"signature": refund.Signature},
				bson.M{"$set": bson.M{"refundStatus": "processing"}},
				true,
			)

			// If the transaction has been confirmed, refund the user
			log.Println("Refunding", refund)
			var signatures []solana.Signature
			if refund.Game == "shuffle" {
				sessionUser, err := shuffle.ParseTransaction(refund.Signature, refund.Token, false, false)
				if err != nil {
					// If the transaction failed to parse, mark it to avoid checking it again
					database.UpdateOne(
						"refund",
						bson.M{"signature": refund.Signature},
						bson.M{"$set": bson.M{"refundStatus": "invalid"}},
						true,
					)
					log.Println(err)
					return
				}
				signatures, err = sendRefundForShuffle(sessionUser, refund.Token)
			} else if refund.Game == "tower" {
				game, err := tower.ParseTransaction(refund.Signature, refund.Token, false, false)
				if err != nil {
					// If the transaction failed to parse, mark it to avoid checking it again
					database.UpdateOne(
						"refund",
						bson.M{"signature": refund.Signature},
						bson.M{"$set": bson.M{"refundStatus": "invalid"}},
						true,
					)
					log.Println(err)
					return
				}
				signatures, err = sendRefundForTower(game, refund.Token)
			}
			if err != nil {
				// If the refund failed, mark it to avoid sending duplicate refunds
				database.UpdateOne(
					"refund",
					bson.M{"signature": refund.Signature},
					bson.M{"$set": bson.M{"refundStatus": "failed"}},
					true,
				)
				log.Println(err)
			}
			database.UpdateOne(
				"refund",
				bson.M{"signature": refund.Signature},
				bson.M{"$set": bson.M{"refundStatus": "sent", "refundSignatures": signatures}},
				true,
			)
		}
		time.Sleep(time.Second * 30)
	}
}
