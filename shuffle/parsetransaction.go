package shuffle

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/csolana"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/vsolana"
)

var (
	signatureMap  = make(map[solana.Signature]struct{})
	signatureLock = sync.Mutex{}
)

type parsedSystemTransferData struct {
	Info struct {
		Destination solana.PublicKey `json:"destination"`
		Lamports    int              `json:"lamports"`
		Source      solana.PublicKey `json:"source"`
	} `json:"info"`
	Type string `json:"type"`
}

type parsedTokenTransferData struct {
	Info struct {
		Authority   solana.PublicKey `json:"authority"`
		Destination solana.PublicKey `json:"destination"`
		Source      solana.PublicKey `json:"source"`
		Amount      string           `json:"amount"`
		TokenAmount struct {
			Amount         string `json:"amount"`
			Decimals       int    `json:"decimals"`
			UIAmount       int    `json:"uiAmount"`
			UIAmountString string `json:"uiAmountString"`
		} `json:"tokenAmount"`
	} `json:"info"`
	Type string `json:"type"`
}

type parsedTransfer struct {
	TokenMint solana.PublicKey

	DestinationOwner solana.PublicKey
	SourceOwner      solana.PublicKey
	Amount           int

	done bool
}

func ParseTransaction(signature solana.Signature, token database.Token, dedup bool, checkTime bool) (SessionUser, error) {
	if dedup {
		signatureLock.Lock()
		_, ok := signatureMap[signature]
		if ok {
			signatureLock.Unlock()
			return SessionUser{}, errors.New("signature already used")
		}
		signatureMap[signature] = struct{}{}
		signatureLock.Unlock()
	}

	sessionUser := SessionUser{
		Signatures: []solana.Signature{signature},
	}

	var transactionData vsolana.TransactionResponse
	for {
		var err error
		transactionData, err = vsolana.GetTransaction(signature)
		if err == nil {
			break
		}
		if err.Error() != "not found" {
			return SessionUser{}, err
		}
		time.Sleep(time.Millisecond * 100)
	}
	if checkTime {
		if time.Now().Unix()-int64(transactionData.Result.Blocktime) > 120 {
			return SessionUser{}, errors.New("transaction too old")
		}
	}

	if token.PublicKey == solana.SolMint {
		solTransfers := parseSolanaTransfers(transactionData)
		// sort by ascending order
		sort.Slice(solTransfers, func(i, j int) bool {
			return solTransfers[i].Amount < solTransfers[j].Amount
		})

		var mints []solana.PublicKey
		for _, postTokenBalance := range transactionData.Result.Meta.Posttokenbalances {
			if postTokenBalance.Owner != env.House().PublicKey() || postTokenBalance.Uitokenamount.Uiamount != 1 {
				continue
			}

			mints = append(mints, postTokenBalance.Mint)
		}

		if len(mints) > 0 {
			var nfts []csolana.NFT
			client := csolana.NewClient(csolana.ClientConfig{
				Endpoint: env.GetRPCUrl(),
			})
			nfts, err := client.GetMultipleNFTsByMint(context.TODO(), mints, &csolana.GetMultipleNFTsByMintOpts{
				IncludeMetadata: true,
				Graceful:        true,
			})
			if err != nil {
				nfts = []csolana.NFT{}
				for _, mint := range mints {
					nft, err := client.GetNFTByMint(context.TODO(), mint, nil)
					if err != nil {
						continue
					}

					nfts = append(nfts, nft)
				}
			}

			var wg sync.WaitGroup
			wg.Add(len(nfts))
			for _, nft := range nfts {
				vnft := vsolana.NFT{
					Mint:        nft.TokenMetadata.Mint,
					MetadataURL: nft.TokenMetadata.Data.Uri,
				}
				go func(wg *sync.WaitGroup, nft vsolana.NFT) {
					defer wg.Done()
					err = nft.Hydrate()
					if err != nil {
						nft.Price = 0
					}
					sessionUser.Assets = append(sessionUser.Assets,
						GeneralAsset{
							Type: "NFT",

							Price: nft.Price,
							Mint:  nft.Mint,

							HadeswapMarket:   nft.HadeswapMarket,
							CollectionSymbol: nft.CollectionSymbol,
							MetadataURL:      nft.MetadataURL,
						},
					)
				}(&wg, vnft)
			}
			wg.Wait()

			sessionUser.Value = sessionUser.Assets.Value()
		}
		// invalid transaction only when there's no nft in the bet
		if len(solTransfers) < 2 && len(mints) == 0 {
			return SessionUser{}, errors.New("invalid transaction, len solTransfers < 2")
		}

		assetTransfer := solTransfers[len(solTransfers)-1]
		// If the last transfer is to the house, then there's sol in the bet
		if assetTransfer.DestinationOwner == env.House().PublicKey() {
			sessionUser.Assets = append(sessionUser.Assets, GeneralAsset{
				Type: "Token",

				Price: assetTransfer.Amount,
				Mint:  solana.SolMint,
			})
		} else if len(mints) == 0 {
			return SessionUser{}, errors.New("invalid transaction, assetTransfer destinationOwner != house")
		}
		feeTransfers := solTransfers
		// If the last transfer is to the house, then it's not a fee transfer
		if assetTransfer.DestinationOwner == env.House().PublicKey() {
			feeTransfers = solTransfers[:len(solTransfers)-1]
		}
		for _, feeTransfer := range feeTransfers {
			sessionUser.feeTransfers = append(sessionUser.feeTransfers, feeTransfer)
			sessionUser.Fee += feeTransfer.Amount
		}
		sessionUser.PublicKey = assetTransfer.SourceOwner
		sessionUser.Value = sessionUser.Assets.Value()

		return sessionUser, nil
	}

	// token is not SOL
	tokenTransfers := parseTokenTransfers(transactionData, token)
	// sort in ascending order to identify fee transaction
	sort.Slice(tokenTransfers, func(i, j int) bool {
		return tokenTransfers[i].Amount < tokenTransfers[j].Amount
	})

	if len(tokenTransfers) < 2 {
		return SessionUser{}, errors.New("invalid transaction, tokenTransfers < 2")
	}

	assetTransfer := tokenTransfers[len(tokenTransfers)-1]
	if assetTransfer.DestinationOwner != env.House().PublicKey() {
		return SessionUser{}, errors.New("invalid transaction, assetTransfer destinationOwner != house")
	}
	sessionUser.Assets = append(sessionUser.Assets, GeneralAsset{
		Type: "Token",

		Price: assetTransfer.Amount,
		Mint:  token.PublicKey,
	})
	feeTransfers := tokenTransfers[:len(tokenTransfers)-1]
	for _, feeTransfer := range feeTransfers {
		sessionUser.feeTransfers = append(sessionUser.feeTransfers, feeTransfer)
		sessionUser.Fee += feeTransfer.Amount
	}
	sessionUser.PublicKey = assetTransfer.SourceOwner
	sessionUser.Value = sessionUser.Assets.Value()

	return sessionUser, nil
}

func parseSolanaTransfers(transactionData vsolana.TransactionResponse) []parsedTransfer {
	var parsedSlice []parsedTransfer

	for _, instruction := range transactionData.Result.Transaction.Message.Instructions {
		if instruction.Programid != solana.SystemProgramID {
			continue
		}

		j, err := json.Marshal(instruction.Parsed)
		if err != nil {
			continue
		}
		var parsed parsedSystemTransferData
		err = json.Unmarshal(j, &parsed)
		if err != nil || parsed.Type != "transfer" {
			continue
		}

		parsedSlice = append(parsedSlice, parsedTransfer{
			TokenMint: solana.SolMint,

			SourceOwner:      parsed.Info.Source,
			DestinationOwner: parsed.Info.Destination,
			Amount:           parsed.Info.Lamports,
		})
	}

	return parsedSlice
}

func parseTokenTransfers(transactionData vsolana.TransactionResponse, token database.Token) []parsedTransfer {
	var parsedSlice []parsedTransfer

	tokenAccountOwnerMap := make(map[solana.PublicKey]solana.PublicKey)

	for _, postTokenBalance := range transactionData.Result.Meta.Posttokenbalances {
		if postTokenBalance.Mint != token.PublicKey {
			continue
		}
		if postTokenBalance.Accountindex > len(transactionData.Result.Transaction.Message.Accountkeys)-1 {
			continue
		}

		account := transactionData.Result.Transaction.Message.Accountkeys[postTokenBalance.Accountindex]

		tokenAccountOwnerMap[account.PublicKey] = postTokenBalance.Owner
	}

	for _, instruction := range transactionData.Result.Transaction.Message.Instructions {
		if instruction.Programid != solana.TokenProgramID {
			continue
		}

		j, err := json.Marshal(instruction.Parsed)
		if err != nil {
			continue
		}
		var parsed parsedTokenTransferData
		err = json.Unmarshal(j, &parsed)
		if err != nil || (parsed.Type != "transfer" && parsed.Type != "transferChecked") {
			continue
		}

		sourceOwner, ok := tokenAccountOwnerMap[parsed.Info.Source]
		if !ok {
			continue
		}
		destinationOwner, ok := tokenAccountOwnerMap[parsed.Info.Destination]
		if !ok {
			continue
		}
		amount, err := strconv.Atoi(parsed.Info.TokenAmount.Amount)
		if err != nil {
			amount, err = strconv.Atoi(parsed.Info.Amount)
			if err != nil {
				continue
			}
		}

		parsedSlice = append(parsedSlice, parsedTransfer{
			TokenMint: token.PublicKey,

			SourceOwner:      sourceOwner,
			DestinationOwner: destinationOwner,
			Amount:           amount,
		})
	}

	return parsedSlice
}
