package csolana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/near/borsh-go"
)

func (c *Client) GetNFTsByOwner(ctx context.Context, publicKey solana.PublicKey) ([]NFT, error) {
	resp, err := c.rpcClient.GetTokenAccountsByOwner(
		ctx,
		publicKey,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Encoding: solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		return []NFT{}, err
	}

	metaAccounts := []solana.PublicKey{}
	for _, tokenAccount := range resp.Value {
		var tokenAccountData TokenAccountData
		err = json.Unmarshal(tokenAccount.Account.Data.GetRawJSON(), &tokenAccountData)
		if err != nil {
			return []NFT{}, err
		}
		if tokenAccountData.Parsed.Info.Tokenamount.Decimals != 0 || tokenAccountData.Parsed.Info.Tokenamount.Amount != "1" {
			continue
		}
		metaAccount, _, err := solana.FindTokenMetadataAddress(tokenAccountData.Parsed.Info.Mint)
		if err != nil {
			continue
		}
		metaAccounts = append(metaAccounts, metaAccount)
	}

	if len(metaAccounts) == 0 {
		return []NFT{}, err
	}

	accountChunks := ChunkBy(metaAccounts, RPCMaximum)
	responseValues := make([]*rpc.Account, len(metaAccounts))
	var wg sync.WaitGroup
	wg.Add(len(accountChunks))
	for i, chunk := range accountChunks {
		go func(gi int, accounts []solana.PublicKey, wg *sync.WaitGroup) {
			defer wg.Done()
			resp, err := c.rpcClient.GetMultipleAccountsWithOpts(
				ctx,
				accounts,
				&rpc.GetMultipleAccountsOpts{
					Encoding: solana.EncodingJSONParsed,
				},
			)
			if err != nil {
				return
			}

			for i, value := range resp.Value {
				responseValues[gi*RPCMaximum+i] = value
			}
		}(i, chunk, &wg)
	}
	wg.Wait()

	if len(responseValues) != len(metaAccounts) {
		return []NFT{}, errors.New("rpc call failed")
	}

	nfts := []NFT{}
	for _, account := range responseValues {
		if account == nil || account.Data == nil {
			continue
		}
		if len(account.Data.GetBinary()) != MetadataAccountSize {
			continue
		}
		tokenMetadata, err := DeserializeMetadata(account.Data.GetBinary())
		if err != nil {
			continue
		}
		nfts = append(nfts, NFT{
			TokenMetadata: tokenMetadata,
		})
	}

	return nfts, nil
}

type GetNFTByMintOpts struct {
	IncludeExternalMetadata bool
	IncludeMetadata         bool
	Graceful                bool
}

var (
	defaultGetNFTOpts = GetNFTByMintOpts{
		IncludeExternalMetadata: false,
		IncludeMetadata:         true,
		Graceful:                false,
	}
	defaultGetMultipleNFTsOpts = GetMultipleNFTsByMintOpts(defaultGetNFTOpts)
)

type GetMultipleNFTsByMintOpts GetNFTByMintOpts

func (c *Client) GetNFTByMint(ctx context.Context, mint solana.PublicKey, opts *GetNFTByMintOpts) (NFT, error) {
	getMultipleNFTsOpts := GetMultipleNFTsByMintOpts(*opts)
	nfts, err := c.GetMultipleNFTsByMint(ctx, []solana.PublicKey{mint}, &getMultipleNFTsOpts)
	if err != nil {
		return NFT{}, err
	}
	return nfts[0], nil
}

func (c *Client) GetMultipleNFTsByMint(ctx context.Context, mints []solana.PublicKey, opts *GetMultipleNFTsByMintOpts) ([]NFT, error) {
	multiplier := 1
	if opts == nil {
		opts = &defaultGetMultipleNFTsOpts
	}

	if opts.IncludeMetadata {
		multiplier = 2
	}

	accounts := make([]solana.PublicKey, len(mints)*multiplier)
	for i, mint := range mints {
		if opts != nil {
			if opts.IncludeMetadata {
				metaAccount, _, err := solana.FindTokenMetadataAddress(mint)
				if err != nil {
					return []NFT{}, fmt.Errorf("invalid mints %e", err)
				}
				accounts[i*2+1] = metaAccount
			}
		}
		accounts[i*multiplier] = mint
	}

	accountChunks := ChunkBy(accounts, RPCMaximum)
	responseValues := make([]*rpc.Account, len(accounts))
	var wg sync.WaitGroup
	wg.Add(len(accountChunks))
	for i, chunk := range accountChunks {
		go func(gi int, accounts []solana.PublicKey, wg *sync.WaitGroup) {
			defer wg.Done()
			resp, err := c.rpcClient.GetMultipleAccountsWithOpts(
				ctx,
				accounts,
				&rpc.GetMultipleAccountsOpts{
					Encoding: solana.EncodingJSONParsed,
				},
			)
			if err != nil {
				return
			}

			for i, value := range resp.Value {
				responseValues[gi*RPCMaximum+i] = value
			}
		}(i, chunk, &wg)
	}
	wg.Wait()

	if len(responseValues) != len(accounts) {
		return []NFT{}, errors.New("rpc call failed")
	}

	nfts := make([]*NFT, len(mints))
	skipIndeces := []int{}

Outer:
	for i, account := range responseValues {
		if account == nil || account.Data == nil {
			if opts.Graceful {
				continue
			}
			return []NFT{}, fmt.Errorf("invalid mint %s %w", mints[i/multiplier].String(), errors.New("account data nil"))
		}

		if opts.IncludeMetadata {
			if i%2 == 1 {
				// is a metadata account
				if len(account.Data.GetBinary()) != MetadataAccountSize {
					if opts.Graceful {
						continue
					}
					return []NFT{}, fmt.Errorf("invalid mint %s %w", mints[i/multiplier].String(), errors.New("invalid account length"))
				}
				tokenMetadata, err := DeserializeMetadata(account.Data.GetBinary())
				if err != nil {
					if opts.Graceful {
						continue
					}
					return []NFT{}, err
				}
				index := i / multiplier
				for _, skip := range skipIndeces {
					if index == skip {
						continue Outer
					}
				}
				nfts[index].TokenMetadata = tokenMetadata
				continue
			}
		}

		// is not a metadata account
		var tokenAccountData TokenAccountData
		err := json.Unmarshal(account.Data.GetRawJSON(), &tokenAccountData)
		if err != nil {
			if opts.Graceful {
				skipIndeces = append(skipIndeces, i/2)
				continue
			}
			return []NFT{}, fmt.Errorf("invalid mint %s %w", mints[i/multiplier].String(), err)
		}
		if tokenAccountData.Parsed.Info.Decimals != 0 || tokenAccountData.Parsed.Info.Supply != "1" {
			if opts.Graceful {
				skipIndeces = append(skipIndeces, i/2)
				continue
			}
			return []NFT{}, fmt.Errorf("invalid mint %s %w", mints[i/multiplier].String(), errors.New("not a non-fungible token"))
		}
		index := i / multiplier
		nfts[index] = &NFT{}
		nfts[index].TokenMetadata.Mint = mints[index]
	}

	var cleanNfts []NFT
	for _, nft := range nfts {
		if nft != nil {
			cleanNfts = append(cleanNfts, *nft)
		}
	}

	return cleanNfts, nil
}

func DeserializeMetadata(data []byte) (TokenMetadata, error) {
	var tokenMetadata TokenMetadata
	err := borsh.Deserialize(&tokenMetadata, data)
	if err != nil {
		return TokenMetadata{}, err
	}
	tokenMetadata.Data.Name = strings.TrimRight(tokenMetadata.Data.Name, "\x00")
	tokenMetadata.Data.Symbol = strings.TrimRight(tokenMetadata.Data.Symbol, "\x00")
	tokenMetadata.Data.Uri = strings.TrimRight(tokenMetadata.Data.Uri, "\x00")
	return tokenMetadata, nil
}
