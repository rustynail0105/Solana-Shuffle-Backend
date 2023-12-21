package shuffle

import (
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/utility"
	"github.com/solanashuffle/backend/vsolana"
)

func (assets Assets) IsEqual(assets2 Assets) bool {
	if len(assets) != len(assets2) {
		return false
	}

	assetMap := make(map[solana.PublicKey]int)
	for _, asset := range assets {
		assetMap[asset.Mint] = asset.Value()
	}

	for _, asset2 := range assets2 {
		value, ok := assetMap[asset2.Mint]
		if !ok || value != asset2.Value() {
			return false
		}
	}

	return true
}

func (assets Assets) SendAndAwaitConfirmation(fromAccount solana.PrivateKey, toAccount solana.PublicKey) ([]solana.Signature, error) {
	if len(assets) == 0 {
		return []solana.Signature{}, nil
	}

	var cleanAssets Assets

	var totalTokenAmount = 0
	var tokenMint solana.PublicKey
	for _, asset := range assets {
		if asset.IsToken() {
			totalTokenAmount += asset.Value()
			tokenMint = asset.Mint
			continue
		} else {
			cleanAssets = append(cleanAssets, asset)
		}
	}

	if totalTokenAmount > 0 {
		cleanAssets = append(cleanAssets, GeneralAsset{
			Type: "Token",

			Mint:  tokenMint,
			Price: totalTokenAmount,
		})
	}

	chunks := utility.ChunkBy(cleanAssets, 5)

	var wg sync.WaitGroup
	wg.Add(len(chunks))

	var signatures []solana.Signature
	for _, assets := range chunks {
		go func(wg *sync.WaitGroup, assets Assets) {
			defer wg.Done()

			var totalInstructions []solana.Instruction
			for _, asset := range assets {
				instructions, err := asset.TransferInstructions(fromAccount, toAccount)
				if err != nil {
					continue
				}

				totalInstructions = append(totalInstructions, instructions...)
			}

			signature, err := vsolana.EnsureInstructions([]solana.PrivateKey{fromAccount}, env.House(), totalInstructions)
			if err != nil {
				return
			}

			signatures = append(signatures, signature)
		}(&wg, assets)
	}

	wg.Wait()

	return signatures, nil
}

func (assets Assets) Value() int {
	var value int
	for _, asset := range assets {
		value += asset.Value()
	}

	return value
}

func (generalAsset GeneralAsset) Value() int {
	return generalAsset.Price
}

func (generalAsset GeneralAsset) TransferInstructions(fromAccount solana.PrivateKey, toAccount solana.PublicKey) ([]solana.Instruction, error) {
	if generalAsset.IsNFT() {
		return vsolana.SendNFTInstructions(
			fromAccount,
			toAccount,
			generalAsset.Mint,
		)
	}

	if generalAsset.Mint == solana.SolMint {
		return vsolana.SendSOLInstructions(
			fromAccount,
			toAccount,
			generalAsset.Price,
		), nil
	}

	return vsolana.CreateAccountAndSendTokenInstructions(fromAccount, toAccount, generalAsset.Mint, generalAsset.Price)
}

func (generalAsset GeneralAsset) IsToken() bool {
	return generalAsset.Type == "Token"
}

func (generalAsset GeneralAsset) IsNFT() bool {
	return generalAsset.Type == "NFT"
}
