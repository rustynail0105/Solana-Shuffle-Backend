package vsolana

import (
	"github.com/gagliardetto/solana-go"
)

const (
	RPCURL    = "https://lively-cool-hill.solana-mainnet.quiknode.pro/c9cdc92c17469a3cc71f79fbbdbf9f6fa6d973e8/"
	HELIUSURL = "https://rpc.helius.xyz/?api-key=edbe04f3-8e44-4c05-8c76-94c6b73ad974"

	MAGICEDENRPC   = "https://api-mainnet.magiceden.io/rpc"
	MAGICEDENSTATS = "https://stats-mainnet.magiceden.io"
	MAGICEDENIDXV2 = "https://api-mainnet.magiceden.io/idxv2"

	LamportsPerSOL = 1_000_000_000

	PublicKeyLength = 32
	MaxSeedLength   = 32
	MaxSeed         = 16

	MaxAccountRequest = 100

	MetaDataLength = 679
)

var (
	SystemProgramID                    = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	TokenProgramID                     = solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	SPLAssociatedTokenAccountProgramID = solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL")
	SPLNameServiceProgramID            = solana.MustPublicKeyFromBase58("namesLPneVptA9Z5rqUDD9tMTWEJwofgaYwp8cawRkX")
	MetaplexTokenMetaProgramID         = solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")
)

func publicKeyFromBytes(b []byte) solana.PublicKey {
	var pubkey solana.PublicKey
	if len(b) > PublicKeyLength {
		b = b[:PublicKeyLength]
	}
	copy(pubkey[PublicKeyLength-len(b):], b)
	return pubkey
}

func btoi64(val []byte) uint64 {
	r := uint64(0)
	for i := uint64(0); i < 8; i++ {
		r |= uint64(val[i]) << (8 * i)
	}
	return r
}

func i64tob(val uint64) []byte {
	r := make([]byte, 8)
	for i := uint64(0); i < 8; i++ {
		r[i] = byte((val >> (i * 8)) & 0xff)
	}
	return r
}
