package shuffle

import (
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/vsolana"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	feeGraceAmount = 500_000
)

func (r *Room) Join(signature solana.Signature) (SessionUser, error) {
	sessionUser, _, err := r.Session.join(signature, r.Token, r.Creator, r.CreatorFeeBasisPoints, r.Config)
	if err != nil {
		return SessionUser{}, err
	}

	r.stream.PublishJSON(fiber.Map{
		"type": "newUser",
		"value": fiber.Map{
			"users":   r.Session.Users,
			"newUser": sessionUser,
			"value":   r.Session.CalculateValue(),
		},
		"id": r.Session.ID,
	})

	return sessionUser, nil
}

func (s *Session) refund(sessionUser SessionUser, token database.Token, errorMessage string) (SessionUser, int, error) {
	// Do not refund the fees for now, as this will slowly drain the shuffle account 
	// sessionUser.Assets = append(sessionUser.Assets, GeneralAsset{
	// 	Type: "Token",
	// 	Price: sessionUser.Assets.Value() * env.FeeBasisPoints() / 10_000,
	// 	Mint:  token.PublicKey,
	// })
	signatures, err := sessionUser.Assets.SendAndAwaitConfirmation(
		env.House(),
		sessionUser.PublicKey,
	)
	if err != nil {
		return SessionUser{}, 0, err
	}

	s.RefundSignatures = append(s.RefundSignatures, signatures...)

	return SessionUser{}, 0, fmt.Errorf("%s - assets being sent back - %s", errorMessage, signatures)
}

func (s *Session) join(signature solana.Signature, token database.Token, creator solana.PublicKey, creatorFeeBasisPoints int, roomConfig RoomConfig) (SessionUser, int, error) {
	atomic.AddInt32(&s.usersOnHold, 1)
	defer func() {
		atomic.AddInt32(&s.usersOnHold, -1)
	}()

	err := vsolana.AwaitConfirmedTransaction(signature)
	if err != nil {
		log.Println(err)
		var refund database.Refund
		err = database.FindOne("refund", bson.M{"signature": signature}, &refund)
		// Only refund if the transaction has not been refunded before
		if err != nil {
			database.InsertOne("refund", database.Refund{
				Signature:    signature,
				Token:        token,
				CreationTime: time.Now().Unix(),
				RefundStatus: "pending",
				Game:         "shuffle",
			})
		}
		return SessionUser{}, 0, errors.New("transaction failed")
	}

	sessionUser, err := ParseTransaction(signature, token, true, true)
	if err != nil {
		return SessionUser{}, 0, err
	}

	if token.PublicKey != solana.SolMint {
		for _, asset := range sessionUser.Assets {
			if asset.Type == "NFT" {
				return SessionUser{}, 0, errors.New("cannot bet NFTs in token room")
			}
		}
	}

	thisBetAmount := sessionUser.Assets.Value()

	if !s.IsInternallyOpen() {
		return s.refund(sessionUser, token, "room not open")
	}

	// Send Bet Webhook
	//utility.SendWebhook(thisBetAmount)

	var requiredTransfers []parsedTransfer
	if env.FeeBasisPoints() > 0 {
		requiredTransfers = append(requiredTransfers, parsedTransfer{
			TokenMint: token.PublicKey,

			DestinationOwner: env.Fee(),
			Amount:           sessionUser.Assets.Value() * env.FeeBasisPoints() / 10_000,
		})
	}

	if creatorFeeBasisPoints > 0 {
		requiredTransfers = append(requiredTransfers, parsedTransfer{
			TokenMint: token.PublicKey,

			DestinationOwner: creator,
			Amount:           sessionUser.Assets.Value() * creatorFeeBasisPoints / 10_000,
		})
	}

	for _, feeTransfer := range sessionUser.feeTransfers {
		for i, requiredTransfer := range requiredTransfers {
			if requiredTransfer.done {
				continue
			}
			if feeTransfer.DestinationOwner == requiredTransfer.DestinationOwner {
				requiredTransfers[i].done = requiredTransfer.Amount-feeTransfer.Amount < feeGraceAmount
			}
		}
	}

	for _, requiredTransfer := range requiredTransfers {
		if !requiredTransfer.done {
			return SessionUser{}, 0, errors.New("fees not paid")
		}
	}

	if err := roomConfig.CheckBetAmount(sessionUser.Value); err != nil {
		return SessionUser{}, 0, err
	}

	var userProfile database.UserProfile
	err = database.FindOne("users", bson.M{"publicKey": sessionUser.PublicKey}, &userProfile)
	if err == nil {
		sessionUser.Profile = SessionProfile{
			Name:  userProfile.Name,
			Image: userProfile.Image,
		}
	}

	var found bool
	for i, user := range s.Users {
		if user.PublicKey != sessionUser.PublicKey {
			continue
		}
		found = true
		// Refund if bet amount exceed max bet amount
		if err := roomConfig.CheckBetAmount(sessionUser.Assets.Value() + s.Users[i].Assets.Value()); err != nil {
			return s.refund(sessionUser, token, "bet amount exceed max bet amount")
		}
		sessionUser.Assets = append(s.Users[i].Assets, sessionUser.Assets...)
		sessionUser.Signatures = append(s.Users[i].Signatures, sessionUser.Signatures...)
		sessionUser.Fee += sessionUser.Fee

		s.Users[i] = &sessionUser
	}

	if !found {
		s.Users = append(s.Users, &sessionUser)
	}

	sessionUser.Value = sessionUser.Assets.Value()

	return sessionUser, thisBetAmount, nil
}

/*

func (r *Room) Join2(publicKey solana.PublicKey) (SessionUser, error) {
	sessionUser, err := r.Session.join2(publicKey, r.Token, r.Creator, r.CreatorFeeBasisPoints, r.Config)
	if err != nil {
		return SessionUser{}, err
	}

	var userProfile database.UserProfile
	err = database.FindOne("users", bson.M{"publicKey": sessionUser.PublicKey}, &userProfile)
	if err == nil {
		sessionUser.Profile = SessionProfile{
			Name:  userProfile.Name,
			Image: userProfile.Image,
		}
	}

	r.stream.PublishJSON(fiber.Map{
		"type": "newUser",
		"value": fiber.Map{
			"users":   r.Session.Users,
			"newUser": sessionUser,
			"value":   r.Session.CalculateValue(),
		},
		"id": r.Session.ID,
	})

	return SessionUser{}, nil
}

func (r *Room) InitJoin2(publicKey solana.PublicKey) (solana.PublicKey, error) {
	return r.Session.initJoin2(publicKey)
}

func (s *Session) initJoin2(publicKey solana.PublicKey) (solana.PublicKey, error) {
	if intermediary, ok := s.getIntermediary(publicKey); ok {
		return intermediary.PublicKey(), nil
	}

	intermediary, err := solana.NewRandomPrivateKey()
	if err != nil {
		return solana.PublicKey{}, err
	}

	s.setIntermediary(publicKey, intermediary)

	return intermediary.PublicKey(), nil
}

func (s *Session) join2(publicKey solana.PublicKey, token Token, creator solana.PublicKey, creatorFeeBasisPoints int, roomConfig RoomConfig) (SessionUser, error) {
	atomic.AddInt32(&s.usersOnHold, 1)
	defer func() {
		atomic.AddInt32(&s.usersOnHold, -1)
	}()

	intermediary, ok := s.getIntermediary(publicKey)
	if !ok {
		return SessionUser{}, errors.New("you have not initialized an intermediary account")
	}

	sessionUser := SessionUser{
		PublicKey: publicKey,
	}

	for _, user := range s.Users {
		if user.PublicKey == sessionUser.PublicKey {
			sessionUser = *user
		}
	}

	iterations := 60 // around 30 seconds of searching
	newAssets, err := sessionUser.ParseIntermediary(intermediary.PublicKey(), token, iterations, rpc.CommitmentConfirmed)
	if err != nil {
		return SessionUser{}, err
	}

	sessionUser.Value = sessionUser.Assets.Value()

	if !s.IsInternallyOpen() {
		fmt.Println(newAssets)
		signatures, err := newAssets.SendAndAwaitConfirmation(
			intermediary,
			sessionUser.PublicKey,
		)
		if err != nil {
			f, err := os.OpenFile("transferErrors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return SessionUser{}, fmt.Errorf("room not open - assets have to be sent back manually, please contact support and take a screenshot of this error: %s", err.Error())
			}
			defer f.Close()
			log.SetOutput(f)
			// log.Println(fmt.Sprintf("err: %s | user: %s | assets: %v", err.Error(), sessionUser.PublicKey.String(), sessionUser.Assets) + "\n")
			return SessionUser{}, fmt.Errorf("room not open - assets have to be sent back manually, please contact support and take a screenshot of this error: %s", err.Error())
		}

		s.RefundSignatures = append(s.RefundSignatures, signatures...)

		return SessionUser{}, fmt.Errorf("room not open - assets being sent back - %s", signatures)
	}

	if err := roomConfig.CheckBetAmount(sessionUser.Value); err != nil {
		return SessionUser{}, err
	}

	// go utility.SendWebhook(sessionUser.Value)
	// does not account for other assets yet
	// SOL not formatted properly

	var found bool
	for i, user := range s.Users {
		if user.PublicKey != sessionUser.PublicKey {
			continue
		}

		found = true
		s.Users[i] = &sessionUser
		break
	}

	if !found {
		s.Users = append(s.Users, &sessionUser)
	}

	return sessionUser, nil
}

*/
