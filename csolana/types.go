package csolana

import (
	"github.com/gagliardetto/solana-go"
	"github.com/near/borsh-go"
)

type NFT struct {
	Metadata      *Metadata     `json:"metadata,omitempty" bson:"metadata"`
	TokenMetadata TokenMetadata `json:"tokenMetadata" bson:"tokenMetadata"`
}

type TokenAccountData struct {
	Parsed struct {
		Info struct {
			Isnative    bool             `json:"isNative"`
			Mint        solana.PublicKey `json:"mint"`
			Owner       solana.PublicKey `json:"owner"`
			State       string           `json:"state"`
			Tokenamount struct {
				Amount         string  `json:"amount"`
				Decimals       int     `json:"decimals"`
				Uiamount       float64 `json:"uiAmount"`
				Uiamountstring string  `json:"uiAmountString"`
			} `json:"tokenAmount"`
			Decimals int    `json:"decimals"`
			Supply   string `json:"supply"`
		} `json:"info"`
		Type string `json:"type"`
	} `json:"parsed"`
	Program string `json:"program"`
	Space   int    `json:"space"`
}

type TokenData struct {
	Parsed struct {
		Info struct {
			Decimals        int    `json:"decimals"`
			Freezeauthority string `json:"freezeAuthority"`
			Isinitialized   bool   `json:"isInitialized"`
			Mintauthority   string `json:"mintAuthority"`
			Supply          string `json:"supply"`
		} `json:"info"`
		Type string `json:"type"`
	} `json:"parsed"`
	Program string `json:"program"`
	Space   int    `json:"space"`
}

type TokenMetadata struct {
	Key             borsh.Enum       `json:"key"`
	UpdateAuthority solana.PublicKey `json:"updateAuthority"`
	Mint            solana.PublicKey `json:"mint"`
	Data            struct {
		Name                 string     `json:"name"`
		Symbol               string     `json:"symbol"`
		Uri                  string     `json:"uri"`
		SellerFeeBasisPoints uint16     `json:"sellerFeeBasisPoints"`
		Creators             *[]Creator `json:"creators"`
	} `json:"data"`
	PrimarySaleHappened bool   `json:"primarySaleHappened"`
	IsMutable           bool   `json:"isMutable"`
	EditionNonce        *uint8 `json:"editionNonce"`
}

type Metadata struct {
	Name                 string `json:"name"`
	Symbol               string `json:"symbol"`
	Description          string `json:"description"`
	SellerFeeBasisPoints int    `json:"seller_fee_basis_points"`
	Image                string `json:"image"`
	AnimationURL         string `json:"animation_url"`
	ExternalURL          string `json:"external_url"`
	Attributes           []struct {
		TraitType string      `json:"trait_type"`
		Value     interface{} `json:"value"`
	} `json:"attributes"`
	Collection struct {
		Name   string `json:"name"`
		Family string `json:"family"`
	} `json:"collection"`
	Properties struct {
		Files []struct {
			URI  string `json:"uri"`
			Type string `json:"type"`
			Cdn  bool   `json:"cdn,omitempty"`
		} `json:"files"`
		Category string `json:"category"`
		Creators []struct {
			Address string `json:"address"`
			Share   int    `json:"share"`
		} `json:"creators"`
	} `json:"properties"`
}

type Creator struct {
	Address  solana.PublicKey `json:"address"`
	Verified bool             `json:"verified"`
	Share    uint8            `json:"share"`
}
