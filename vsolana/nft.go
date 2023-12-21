package vsolana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"sync"

	http "github.com/bogdanfinn/fhttp"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/shuffle/price"
	"github.com/solanashuffle/backend/utility"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func (nft *NFT) SendAndAwaitConfirmation(fromAccount solana.PrivateKey, toAccount solana.PublicKey) (solana.Signature, error) {
	instructions, err := SendNFTInstructions(fromAccount, toAccount, nft.Mint)
	if err != nil {
		return solana.Signature{}, err
	}

	return EnsureInstructions([]solana.PrivateKey{fromAccount}, env.House(), instructions)
}

func (nftBatch NFTBatch) SendAndAwaitConfirmation(fromAccount solana.PrivateKey, toAccount solana.PublicKey) ([]solana.Signature, error) {
	if len(nftBatch) == 0 {
		return []solana.Signature{}, nil
	}

	chunks := utility.ChunkBy(nftBatch, 5)

	var wg sync.WaitGroup
	wg.Add(len(chunks))

	var signatures []solana.Signature
	for _, nftBatch := range chunks {
		go func(wg *sync.WaitGroup, nftBatch NFTBatch) {
			defer wg.Done()

			var batchInstructions []solana.Instruction
			for _, nft := range nftBatch {
				instructions, err := SendNFTInstructions(fromAccount, toAccount, nft.Mint)
				if err != nil {
					continue
				}
				batchInstructions = append(batchInstructions, instructions...)
			}

			signature, err := EnsureInstructions([]solana.PrivateKey{fromAccount}, env.House(), batchInstructions)
			if err != nil {
				return
			}

			signatures = append(signatures, signature)
		}(&wg, nftBatch)
	}

	wg.Wait()

	return signatures, nil
}

type getNFTStatsByMintAddressResponse struct {
	Results struct {
		Mintaddress      string `json:"mintAddress"`
		Collectionsymbol string `json:"collectionSymbol"`
		Attrs            []struct {
			TraitType  string `json:"trait_type"`
			Value      string `json:"value"`
			Valuecount int    `json:"valueCount"`
		} `json:"attrs"`
		Totalmints int `json:"totalMints"`
		Raritya    int `json:"rarityA"`
		Ranka      int `json:"rankA"`
	} `json:"results"`
}

func (nft *NFT) GetCollectionSymbol() (string, error) {
	client, err := utility.NewTLSClient()
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/getNFTStatsByMintAddress/%s", MAGICEDENRPC, nft.Mint.String()), nil)
	if err != nil {
		return "", err
	}

	req.Header = utility.BrowserHeaders()

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var bodyMap map[string]interface{}
	err = json.Unmarshal(body, &bodyMap)
	if err != nil {
		return "", err
	}

	if collectionSymbol, ok := bodyMap["collectionSymbol"]; ok {
		return fmt.Sprint(collectionSymbol), nil
	}

	var parsed getNFTStatsByMintAddressResponse
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return "", err
	}

	if parsed.Results.Collectionsymbol == "" {
		return "", errors.New("no collection found")
	}

	nft.CollectionSymbol = parsed.Results.Collectionsymbol

	return parsed.Results.Collectionsymbol, nil
}

func (nft *NFT) GetCollectionSymbolCache() (string, error) {
	val, err := database.RDB.Get(
		context.TODO(),
		nft.Mint.String(),
	).Result()
	if err == nil {
		nft.CollectionSymbol = val
	}
	return val, err
}

func GetHadeswapMarket(collectionSymbol string) (solana.PublicKey, error) {
	val, err := database.RDB.Get(
		context.TODO(),
		collectionSymbol,
	).Result()
	if err != nil {
		return solana.PublicKey{}, err
	}
	return solana.PublicKeyFromBase58(val)
}

func (nft *NFT) Hydrate() error {
	collectionSymbol, err := nft.GetCollectionSymbolCache()
	if err != nil {
		return err
	}
	price, err := price.Estimate(collectionSymbol)
	if err != nil {
		return err
	}
	nft.Price = price
	hadeswapMarket, err := GetHadeswapMarket(collectionSymbol)
	if err != nil {
		return err
	}
	nft.HadeswapMarket = hadeswapMarket
	return nil
}

func FetchNFTByMint(mint solana.PublicKey) (NFT, error) {
	rpcClient := rpc.New(RPCURL)

	resp, err := rpcClient.GetAccountInfoWithOpts(
		context.TODO(),
		mint,
		&rpc.GetAccountInfoOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		return NFT{}, err
	}

	var tokenData TokenData
	err = json.Unmarshal(resp.Value.Data.GetRawJSON(), &tokenData)
	if err != nil {
		return NFT{}, err
	}
	if tokenData.Parsed.Info.Decimals != 0 || tokenData.Parsed.Info.Supply != "1" {
		return NFT{}, errors.New("not an nft")
	}

	nft := NFT{
		Mint: mint,
	}

	return nft, nil
}

func FetchMultipleNFTsByMint(mints []solana.PublicKey) ([]NFT, error) {
	rpcClient := rpc.New(RPCURL)

	resp, err := rpcClient.GetMultipleAccountsWithOpts(
		context.TODO(),
		mints,
		&rpc.GetMultipleAccountsOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		return []NFT{}, err
	}

	var nfts []NFT
	for i, account := range resp.Value {
		var tokenData TokenData
		err = json.Unmarshal(account.Data.GetRawJSON(), &tokenData)
		if err != nil {
			continue
		}
		if tokenData.Parsed.Info.Decimals != 0 || tokenData.Parsed.Info.Supply != "1" {
			continue
		}
		nfts = append(nfts, NFT{
			Mint: mints[i],
		})
	}

	return nfts, nil
}

func FetchUserNfts(pub solana.PublicKey) ([]NFT, error) {
	client := rpc.New(RPCURL)

	resp, err := client.GetTokenAccountsByOwner(
		context.TODO(),
		pub,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		return []NFT{}, err
	}

	var nfts []NFT
	for _, tokenAccount := range resp.Value {
		var tokenAccountData TokenAccountData
		err = json.Unmarshal(tokenAccount.Account.Data.GetRawJSON(), &tokenAccountData)
		if err != nil {
			return []NFT{}, err
		}
		if tokenAccountData.Parsed.Info.Tokenamount.Decimals != 0 || tokenAccountData.Parsed.Info.Tokenamount.Amount != "1" {
			continue
		}
		mint := solana.MustPublicKeyFromBase58(tokenAccountData.Parsed.Info.Mint)

		nft := NFT{
			Mint: mint,
		}

		nfts = append(nfts, nft)
	}

	return nfts, nil
}
