package tower

import (
	"encoding/json"
	"errors"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/solanashuffle/backend/database"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/vsolana"
)

const (
	transactionAgeLimit = 120 // in seconds
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
}

func ParseTransaction(signature solana.Signature, token database.Token, dedup bool, checkTime bool) (ParsedTransactionData, error) {
	if dedup {
		signatureLock.Lock()
		_, ok := signatureMap[signature]
		if ok {
			signatureLock.Unlock()
			return ParsedTransactionData{}, errors.New("signature already used")
		}
		signatureMap[signature] = struct{}{}
		signatureLock.Unlock()
	}

	var transactionData vsolana.TransactionResponse
	for i := 0; true; i++ {
		var err error
		transactionData, err = vsolana.GetTransaction(signature)
		log.Println(err)
		if err == nil {
			break
		}
		if err.Error() != "not found" && i > 64 {
			return ParsedTransactionData{}, err
		}
		time.Sleep(time.Millisecond * 300)
	}

	if checkTime {
		if time.Now().Unix()-int64(transactionData.Result.Blocktime) > transactionAgeLimit {
			return ParsedTransactionData{}, errors.New("transaction too old")
		}
	}

	var transfers []parsedTransfer

	if token.PublicKey == solana.SolMint {
		// token is SOL
		transfers = parseSolanaTransfers(transactionData)
	} else {
		// token is SPL
		transfers = parseTokenTransfers(transactionData, token)
	}

	// sort transfers in ascending to identify fee transfer
	sort.Slice(transfers, func(i, j int) bool {
		return transfers[i].Amount < transfers[j].Amount
	})

	// transfers length needs to be 2
	// we need 2x value transfer for bet & fee
	if len(transfers) != 2 {
		return ParsedTransactionData{}, errors.New("fee not paid")
	}

	feeTransfer := transfers[0]
	betTransfer := transfers[1]

	if feeTransfer.DestinationOwner != env.Fee() {
		return ParsedTransactionData{}, errors.New("fee sent to wrong address")
	}

	if betTransfer.DestinationOwner != env.TowerHouse().PublicKey() {
		return ParsedTransactionData{}, errors.New("bet sent to wrong address")
	}

	return ParsedTransactionData{
		Signature: signature,
		PublicKey: betTransfer.SourceOwner,

		BetAmount: betTransfer.Amount,
		FeeAmount: feeTransfer.Amount,
	}, nil
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
