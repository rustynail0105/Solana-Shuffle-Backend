package vsolana

import (
	"crypto/ed25519"

	"github.com/gagliardetto/solana-go"
)

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	var _chunks = make([][]T, 0, (len(items)/chunkSize)+1)
	for chunkSize < len(items) {
		items, _chunks = items[chunkSize:], append(_chunks, items[0:chunkSize:chunkSize])
	}
	return append(_chunks, items)
}

func VerifySignature(signature solana.Signature, publicKey solana.PublicKey, message []byte) bool {
	return ed25519.Verify(ed25519.PublicKey(publicKey[:]), message, signature[:])
}
