package fair

// attempts to provide a provably fair number between 0-10_000
// not fully provable yet
// problem:
// based on random hashes of solana blocks
// solana sometimes skips blocks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/gagliardetto/solana-go/rpc"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/utility"
)

func Random() (int, Proof, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	var slotId uint64
	latestSlot, err := rpcClient.GetSlot(context.TODO(), rpc.CommitmentConfirmed)
	if err != nil {
		return 0, Proof{}, err
	}
	slotId = latestSlot

	var block *rpc.GetBlockResult
	for i := 0; i < 10; i++ {
		var err error
		block, err = rpcClient.GetBlockWithOpts(context.TODO(), slotId, &rpc.GetBlockOpts{
			TransactionDetails: rpc.TransactionDetailsNone,
			Rewards:            &utility.False,
			Commitment:         rpc.CommitmentConfirmed,
		})
		if err == nil {
			break
		}
		time.Sleep(time.Millisecond * 500)
		if i == 9 {
			return utility.RandomInt(0, 10000), Proof{}, nil
		}
	}

	signature, err := env.House().Sign(block.Blockhash[:])
	if err != nil {
		return 0, Proof{}, err
	}

	signatureHash := CreateHash(signature[:])

	max := new(big.Int)
	base, exponent := big.NewInt(2), big.NewInt(256)
	max.Exp(base, exponent, nil)

	randomValue := new(big.Int)
	randomValue.SetString(hex.EncodeToString(signatureHash), 16)
	randomValue.Mul(randomValue, big.NewInt(10_000))
	randomValue.Div(randomValue, max)

	return int(randomValue.Int64()),
		Proof{
			BlockSlot:     slotId,
			BlockHash:     block.Blockhash,
			FairSignature: signature,
		}, nil
}

func CreateHash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}
