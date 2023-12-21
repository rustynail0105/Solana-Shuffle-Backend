package shuffle

import (
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/shuffle/fair"
	"github.com/solanashuffle/backend/stream"
)

type Room struct {
	ID string `json:"id" bson:"id"`

	Name  string         `json:"name" bson:"name"`
	Token database.Token `json:"token" bson:"token"`

	CoverImage   string           `json:"coverImage" bson:"coverImage"`
	Creator      solana.PublicKey `json:"creator" bson:"creator"`
	CreationTime int64            `json:"creationTime" bson:"creationTime"`
	Official     bool             `json:"official" bson:"official"`
	Public       bool             `json:"public" bson:"public"`
	Stats        database.Stats   `json:"stats" bson:"stats"`

	CreatorFeeBasisPoints int `json:"creatorFeeBasisPoints" bson:"creatorFeeBasisPoints"`

	Config RoomConfig `json:"config"`

	Session *Session `json:"session" bson:"-"`

	stream *stream.Stream
}

type RoomConfig struct {
	MinimumAmount int `json:"minimumAmount" bson:"minimumAmount"`
	MaximumAmount int `json:"maximumAmount" bson:"maximumAmount"`
}

type RoomHistory struct {
	ID              string `json:"id" bson:"id"`
	UniqueUserCount int    `json:"uniqueUserCount" bson:"uniqueUserCount"`
}

type Session struct {
	RoomID       string        `json:"roomId" bson:"roomId"`
	ID           string        `json:"id" bson:"id"`
	CreationTime int64         `json:"creationTime" bson:"creationTime"`
	CloseTime    int64         `json:"closeTime" bson:"closeTime"`
	Status       string        `json:"status" bson:"-"`
	Countdown    time.Duration `json:"countdown" bson:"-"`

	Users                []*SessionUser                         `json:"users" bson:"users"`
	IntermediaryAccounts map[solana.PublicKey]solana.PrivateKey `json:"-" bson:"intermediaryAccounts"`
	intermediaryMu       *sync.RWMutex

	RefundSignatures []solana.Signature `json:"refundSignatures" bson:"refundSignatures"`

	usersOnHold int32

	Value int `json:"value" bson:"value"`

	Result Result `json:"result" bson:"result"`
}

type Result struct {
	Winner     solana.PublicKey   `json:"winner" bson:"winner"`
	Signatures []solana.Signature `json:"signatures,omitempty" bson:"signatures"`

	Assets Assets `json:"assets" bson:"-"`

	Value int `json:"value" bson:"-"`

	SpinValue int        `json:"spinValue" bson:"spinValue"`
	FairProof fair.Proof `json:"fairProof" bson:"fairProof"`

	Time int64 `json:"time" bson:"time"`
}

type SessionUser struct {
	PublicKey  solana.PublicKey   `json:"publicKey"`
	Signatures []solana.Signature `json:"signatures"`

	Assets Assets `json:"assets" bson:"assets"`

	Value int `json:"value"`

	Fee          int `json:"fee" bson:"fee"`
	feeTransfers []parsedTransfer

	Profile SessionProfile `json:"profile" bson:"-"`
}

type SessionProfile struct {
	Name  string `json:"name" bson:"name"`
	Image string `json:"image" bson:"image"`
}

type GeneralAsset struct {
	Type string `json:"type" bson:"type"`

	Price            int              `json:"price" bson:"price"`
	Mint             solana.PublicKey `json:"mint,omitempty" bson:"mint,omitempty"`
	HadeswapMarket   solana.PublicKey `json:"hadeswapMarket,omitempty" bson:"hadeswapMarket,omitempty"`
	CollectionSymbol string           `json:"collectionSymbol,omitempty" bson:"collectionSymbol,omitempty"`
	MetadataURL      string           `json:"metadataURL" bson:"metadataURL"`
}

type Assets []GeneralAsset
