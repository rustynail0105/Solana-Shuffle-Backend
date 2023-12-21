package shuffle

import (
	"fmt"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/uuid"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/shuffle/conversion"
	"github.com/solanashuffle/backend/stream"
	"github.com/solanashuffle/backend/utility"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

func (r *Room) NewSession() {
	r.Session = &Session{
		ID:     uuid.NewString(),
		RoomID: r.ID,

		Users:                []*SessionUser{},
		IntermediaryAccounts: make(map[solana.PublicKey]solana.PrivateKey),
		intermediaryMu:       &sync.RWMutex{},

		RefundSignatures: []solana.Signature{},

		CreationTime: time.Now().Unix(),
	}
}

func (s *Session) CalculateValue() int {
	var value int
	for _, user := range s.Users {
		value += user.Assets.Value()
	}
	s.Value = value
	return value
}

func (s *Session) Assets() Assets {
	var assets Assets
	for _, user := range s.Users {
		assets = append(assets, user.Assets...)
	}

	return assets
}

func (s *Session) Track(token database.Token) error {
	s.CloseTime = time.Now().Unix()

	for _, sessionUser := range s.Users {
		volume := sessionUser.Assets.Value()
		solAmount := volume
		if token.PublicKey != solana.SolMint {
			var err error
			solAmount, err = conversion.ToSOL(solAmount, token.PublicKey)
			if err != nil {
				solAmount = 0
			}
		}

		database.UpdateOne(
			"users",
			bson.M{"publicKey": sessionUser.PublicKey},
			bson.M{"$inc": bson.M{
				"stats.totalGames":  1,
				"stats.totalVolume": solAmount,
				fmt.Sprintf("stats.games.%s", utility.FormatDate(time.Now())):   1,
				fmt.Sprintf("stats.volumes.%s", utility.FormatDate(time.Now())): solAmount,
			}},
			true,
		)

		database.UpdateOne(
			"rooms",
			bson.M{"id": s.RoomID},
			bson.M{"$inc": bson.M{
				"stats.totalGames":  1,
				"stats.totalVolume": volume,
				fmt.Sprintf("stats.games.%s", utility.FormatDate(time.Now())):   1,
				fmt.Sprintf("stats.volumes.%s", utility.FormatDate(time.Now())): volume,
			}},
			true,
		)
	}

	return database.InsertOne(
		"sessions",
		s,
	)
}

func (s *Session) IsInternallyOpen() bool {
	return s.Status != "drawing" && s.Status != "finished" && s.Countdown > time.Second
}

func (s *Session) IsPubliclyOpen() bool {
	return s.Countdown > time.Second*3
}

func (s *Session) WaitUntilNotOnHold() {
	for {
		if s.usersOnHold == 0 {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (s *Session) WaitUntilAssetsFinalized() {
	var signatures []solana.Signature
	for _, user := range s.Users {
		signatures = append(signatures, user.Signatures...)
	}

	vsolana.AwaitSignatureStatuses(signatures, rpc.ConfirmationStatusFinalized)
}

/*

func (s *Session) WaitUntilAssetsFinalized2(token Token) {
	var signatures []solana.Signature

	rpcClient := rpc.New(env.GetRPCUrl())

	var wg sync.WaitGroup
	for _, sessionUser := range s.Users {
		wg.Add(1)
		go func(wg *sync.WaitGroup, sessionUser *SessionUser) {
			defer wg.Done()
			intermediary, ok := s.getIntermediary(sessionUser.PublicKey)
			if !ok {
				panic(errors.New("sessionUser without intermediary in session"))
			}

			fmt.Println(intermediary.PublicKey(), intermediary)

			limit := 100
			resp, err := rpcClient.GetSignaturesForAddressWithOpts(
				context.TODO(),
				intermediary.PublicKey(),
				&rpc.GetSignaturesForAddressOpts{
					Limit:      &limit,
					Commitment: rpc.CommitmentConfirmed,
				},
			)
			if err != nil {
				return
			}

		Outer:
			for _, tx := range resp {
				if tx.Err != nil {
					continue Outer
				}
				for _, sig := range s.RefundSignatures {
					if sig == tx.Signature {
						continue Outer
					}
				}
				signatures = append(signatures, tx.Signature)
				sessionUser.Signatures = append(sessionUser.Signatures, tx.Signature)
			}
		}(&wg, sessionUser)
	}
	wg.Wait()
	vsolana.AwaitSignatureStatuses(signatures, rpc.ConfirmationStatusFinalized)
}

*/

func (s *Session) WaitUntilPopulated(stream *stream.Stream) {
	updateChannel := stream.Subscribe()
	defer stream.Unsubscribe(updateChannel)
	for {
		if s.IsPopulated() {
			return
		}
		<-updateChannel
	}
}

func (s *Session) IsPopulated() bool {
	return len(s.Users) >= 2
}

/*

func (s *Session) getIntermediary(publicKey solana.PublicKey) (solana.PrivateKey, bool) {
	//s.intermediaryMu.RLock()
	//defer s.intermediaryMu.RUnlock()
	intermediary, ok := s.IntermediaryAccounts[publicKey]
	return intermediary, ok
}

func (s *Session) setIntermediary(publicKey solana.PublicKey, intermediary solana.PrivateKey) {
	//s.intermediaryMu.Lock()
	//defer s.intermediaryMu.Unlock()
	s.IntermediaryAccounts[publicKey] = intermediary
}

*/
