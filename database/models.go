package database

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
)

type Token struct {
	Ticker    string           `json:"ticker" bson:"ticker"`
	PublicKey solana.PublicKey `json:"publicKey" bson:"publicKey"`
	Decimals  int              `json:"decimals" bson:"decimals"`
}

type Refund struct {
	Signature        solana.Signature   `json:"signature" bson:"signature"`
	RefundStatus     string             `json:"refundStatus" bson:"refundStatus"` // "pending", "sent", "failed"
	CreationTime     int64              `json:"creationTime" bson:"creationTime"`
	Token            Token              `json:"token" bson:"token"`
	RefundSignatures []solana.Signature `json:"refundSignatures" bson:"refundSignatures"`
	Game             string             `json:"game" bson:"game"`
}
type UserProfile struct {
	PublicKey solana.PublicKey `json:"publicKey" bson:"publicKey"`
	Signature solana.Signature `json:"signature" bson:"signature"`

	Name  string `json:"name" bson:"name"`
	About string `json:"about" bson:"about"`
	//Balance uint64 `json:"balance" bson:"balance"`
	//Discord Discord `json:"discord" bson:"discord"`
	//Twitter Twitter `json:"twitter" bson:"twitter"`
	Image  string `json:"image" bson:"image"`
	Banner string `json:"banner" bson:"banner"`

	FavoriteRooms []string `json:"favoriteRooms" bson:"favoriteRooms"`

	Stats    Stats    `json:"stats,omitempty" bson:"stats"`
	Referral Referral `json:"referral,omitempty" bson:"referral"`
}

type Referral struct {
	PublicKey solana.PublicKey `json:"publicKey" bson:"publicKey"`
	Signature solana.Signature `json:"signature" bson:"signature"`
	MyLink    string           `json:"myLink" bson:"myLink"`
	PlayLink  string           `json:"playLink" bson:"playLink"`
	Wallet    string           `json:"wallet" bson:"wallet"` // this is the referrer's wallet.
	Earned    uint64           `json:"earned" bson:"earned"`
	Used      bool             `json:"used" bson:"used"`
	Activated time.Time        `json:"activated" bson:"activated"`
}

type Stats struct {
	TotalVolume uint64 `json:"totalVolume" bson:"totalVolume"`
	TotalGames  uint64 `json:"totalGames" bson:"totalGames"`
	TotalWin    uint64 `json:"totalWin" bson:"totalWin"`
	TotalLoss   uint64 `json:"totalLoss" bson:"totalLoss"`
	Level       Level  `json:"level" bson:"level"`

	Volumes map[string]uint64 `json:"volumes" bson:"volumes"`
	Games   map[string]uint64 `json:"games" bson:"games"`
}

type Level struct {
	Value  uint64 `json:"value" bson:"value"`
	XP     uint64 `json:"xp" bson:"xp"`
	Needed uint64 `json:"needed" bson:"needed"`
}

type Discord struct {
	ID            string `json:"id" bson:"id"`
	Username      string `json:"username" bson:"username"`
	Discriminator string `json:"discriminator" bson:"discriminator"`

	Avatar string `json:"avatar" bson:"avatar"`
}

func (d Discord) AvatarURL() string {
	if d.Avatar == "" || d.ID == "" {
		return ""
	}

	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", d.ID, d.Avatar)
}

type Twitter struct {
	ID   string `json:"id" bson:"id"`
	Name string `json:"name" bson:"name"`

	Avatar string `json:"avatar" bson:"avatar"`
}
