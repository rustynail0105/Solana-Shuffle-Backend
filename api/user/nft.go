package user

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gofiber/fiber/v2"
	"github.com/solanashuffle/backend/csolana"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle"
	"github.com/solanashuffle/backend/vsolana"
)

func HandleUserAssetsGET(c *fiber.Ctx) error {
	client := csolana.NewClient(csolana.ClientConfig{
		Endpoint: env.GetRPCUrl(),
	})

	publicKey, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	cnfts, err := client.GetNFTsByOwner(
		context.TODO(),
		publicKey,
	)
	if err != nil {
		return JSONError(c, fiber.StatusInternalServerError, err)
	}

	var assets []shuffle.GeneralAsset
	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	wg.Add(len(cnfts))

	for _, cnft := range cnfts {
		go func(wg *sync.WaitGroup, cnft csolana.NFT) {
			defer wg.Done()
			nft := vsolana.NFT{
				Mint:        cnft.TokenMetadata.Mint,
				MetadataURL: cnft.TokenMetadata.Data.Uri,
			}

			err := nft.Hydrate()
			if err != nil {
				nft.Price = 0
			}

			mu.Lock()
			assets = append(assets, shuffle.GeneralAsset{
				Type:             "NFT",
				Mint:             nft.Mint,
				Price:            nft.Price,
				HadeswapMarket:   nft.HadeswapMarket,
				CollectionSymbol: nft.CollectionSymbol,
				MetadataURL:      nft.MetadataURL,
			})
			mu.Unlock()
		}(&wg, cnft)
	}

	wg.Wait()

	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Price > assets[j].Price
	})

	return c.Status(fiber.StatusOK).JSON(assets)
}

func HandleNFTGET(c *fiber.Ctx) error {
	mint, err := solana.PublicKeyFromBase58(c.Params("+"))
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	nft, err := vsolana.FetchNFTByMint(mint)
	if err != nil {
		return JSONError(c, fiber.StatusBadRequest, err)
	}

	err = nft.Hydrate()
	if err != nil {
		nft.Price = 0
		return c.Status(fiber.StatusOK).JSON(nft)
	}

	return c.Status(fiber.StatusOK).JSON(nft)
}

func HandleNFTBatchGet(c *fiber.Ctx) error {
	mintStrings := strings.Split(c.Params("+"), ",")

	var wg sync.WaitGroup
	var nfts []vsolana.NFT

	wg.Add(len(mintStrings))

	for _, mintString := range mintStrings {
		go func(wg *sync.WaitGroup, mintString string) {
			defer wg.Done()
			mint, err := solana.PublicKeyFromBase58(mintString)
			if err != nil {
				return
			}

			nft, err := vsolana.FetchNFTByMint(mint)
			if err != nil {
				return
			}

			err = nft.Hydrate()
			if err != nil {
				nft.Price = 0
			}

			nfts = append(nfts, nft)
		}(&wg, mintString)
	}

	wg.Wait()

	return c.Status(fiber.StatusOK).JSON(nfts)
}
