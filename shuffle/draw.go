package shuffle

import (
	"fmt"
	"math"
	"time"

	"github.com/solanashuffle/backend/database"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle/fair"
)

func (r *Room) Draw() (Result, chan []solana.Signature, error) {
	return r.Session.draw(r.Token)
}

func (s *Session) draw(token database.Token) (Result, chan []solana.Signature, error) {
	resultAnnotationChannel := make(chan []solana.Signature)

	spinValue, fairProof, err := fair.Random()
	if err != nil {
		return Result{}, resultAnnotationChannel, err
	}

	var maxValue int
	var winner solana.PublicKey
	for _, user := range s.Users {
		maxValue += int(
			math.Ceil(
				float64(user.Assets.Value()*10_000) / float64(s.CalculateValue()),
			),
		)

		if maxValue >= spinValue && winner.IsZero() {
			winner = user.PublicKey
			err := database.UpdateOne(
				"users",
				bson.M{"publicKey": user.PublicKey},
				bson.M{"$inc": bson.M{
					"stats.totalWins": 1,
				}},
				true,
			)
			if err != nil {
				return Result{}, nil, err
			}
			break
		} else {
			err := database.UpdateOne(
				"users",
				bson.M{"publicKey": user.PublicKey},
				bson.M{"$inc": bson.M{
					"stats.totalLoss": 1,
				}},
				true,
			)
			if err != nil {
				return Result{}, nil, err
			}
		}
	}

	result := Result{
		Winner: winner,

		Assets:    s.Assets(),
		FairProof: fairProof,

		SpinValue: spinValue,

		Time: time.Now().Unix(),
	}

	result.Value = result.Assets.Value()

	s.Result = result

	go func(resultAnnotationChannel chan []solana.Signature) {
		s.WaitUntilAssetsFinalized()
		signatures, err := result.Assets.SendAndAwaitConfirmation(
			env.House(),
			winner,
		)
		if err != nil {
			fmt.Println("error sending winnings")
			fmt.Println(winner)
			fmt.Println(err)
			return
		}
		time.Sleep(time.Second * 5)
		fmt.Println("sending...")
		resultAnnotationChannel <- signatures
		return
	}(resultAnnotationChannel)

	return result, resultAnnotationChannel, nil
}

/*
func (s *Session) SendWinnings(winner solana.PublicKey) []solana.Signature {
	var signatures []solana.Signature

	fmt.Println("sending winnings...")

	fmt.Println(len(s.Users))
	var wg sync.WaitGroup
	for _, user := range s.Users {
		fmt.Println(user.Assets)
		fmt.Println(user.Assets.Value())
		if len(user.Assets) == 0 {
			continue
		}
		intermediary, ok := s.getIntermediary(user.PublicKey)
		if !ok {
			continue
		}
		var totalInstructions []solana.Instruction
		for _, asset := range user.Assets {
			instructions, err := asset.TransferInstructions(intermediary, winner)
			if err != nil {
				fmt.Println(err)
				continue
			}

			totalInstructions = append(totalInstructions, instructions...)
		}

		if len(totalInstructions) == 0 {
			fmt.Println("len totalInstructions")
			continue
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, thisInstructions []solana.Instruction, signer solana.PrivateKey) {
			defer wg.Done()
			fmt.Println(signer.PublicKey())
			signature, err := vsolana.EnsureInstructions([]solana.PrivateKey{signer}, env.House(), thisInstructions)
			if err != nil {
				fmt.Println(err)
				f, err := os.OpenFile("winnerErrors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
				if err != nil {
					return
				}
				defer f.Close()
				log.SetOutput(f)
				//log.Println(fmt.Sprintf("err: %s | signer: %s | instructions: %v", err.Error(), signer, thisInstructions) + "\n")
			}
			fmt.Println(signatures)
			signatures = append(signatures, signature)
		}(&wg, totalInstructions, intermediary)
	}

	fmt.Println("awaiting winning tx")
	wg.Wait()
	fmt.Println(signatures)

	return signatures
}
*/
