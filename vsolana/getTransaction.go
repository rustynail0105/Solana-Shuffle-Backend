package vsolana

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/solanashuffle/backend/env"
)

var (
	getConfirmedTransactionThreadDelay = 50 * time.Millisecond
)

type TransactionResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  *struct {
		Blocktime int `json:"blockTime"`
		Meta      struct {
			Err               interface{} `json:"err"`
			Fee               int         `json:"fee"`
			Innerinstructions []struct {
				Index        int `json:"index"`
				Instructions []struct {
					Parsed struct {
						Info struct {
							NewAccount  string `json:"newAccount"`
							Account     string `json:"account"`
							Owner       string `json:"owner"`
							Destination string `json:"destination"`
							Lamports    int    `json:"lamports"`
							Source      string `json:"source"`
						} `json:"info"`
						Type string `json:"type"`
					} `json:"parsed"`
					Program   string `json:"program"`
					Programid string `json:"programId"`
				} `json:"instructions"`
			} `json:"innerInstructions"`
			Logmessages       []string `json:"logMessages"`
			Postbalances      []int    `json:"postBalances"`
			Posttokenbalances []struct {
				Accountindex  int              `json:"accountIndex"`
				Mint          solana.PublicKey `json:"mint"`
				Owner         solana.PublicKey `json:"owner"`
				Uitokenamount struct {
					Amount         string  `json:"amount"`
					Decimals       int     `json:"decimals"`
					Uiamount       float64 `json:"uiAmount"`
					Uiamountstring string  `json:"uiAmountString"`
				} `json:"uiTokenAmount"`
			} `json:"postTokenBalances"`
			Prebalances      []int `json:"preBalances"`
			Pretokenbalances []struct {
				Accountindex  int              `json:"accountIndex"`
				Mint          solana.PublicKey `json:"mint"`
				Owner         solana.PublicKey `json:"owner"`
				Uitokenamount struct {
					Amount         string  `json:"amount"`
					Decimals       int     `json:"decimals"`
					Uiamount       float64 `json:"uiAmount"`
					Uiamountstring string  `json:"uiAmountString"`
				} `json:"uiTokenAmount"`
			} `json:"preTokenBalances"`
			Rewards []interface{} `json:"rewards"`
			Status  struct {
				Ok interface{} `json:"Ok"`
			} `json:"status"`
		} `json:"meta"`
		Slot        int `json:"slot"`
		Transaction struct {
			Message struct {
				Accountkeys []struct {
					PublicKey solana.PublicKey `json:"pubkey"`
					Signer    bool             `json:"signer"`
					Writable  bool             `json:"writable"`
				} `json:"accountKeys"`
				Instructions []struct {
					Parsed    interface{}      `json:"parsed"`
					Data      string           `json:"data"`
					Program   string           `json:"program"`
					Programid solana.PublicKey `json:"programId"`
				} `json:"instructions"`
				Recentblockhash string `json:"recentBlockhash"`
			} `json:"message"`
			Signatures []string `json:"signatures"`
		} `json:"transaction"`
	} `json:"result"`
	ID string `json:"id"`
}

func GetTransaction(signature solana.Signature) (TransactionResponse, error) {
	type requestStruct struct {
		ID      string        `json:"id"`
		JsonRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	type configStruct struct {
		Encoding                       string `json:"encoding"`
		Commitment                     string `json:"commitment"`
		MaxSupportedTransactionVersion int    `json:"maxSupportedTransactionVersion"`
	}

	payload := requestStruct{
		ID:      string(uuid.NewString()),
		Method:  "getTransaction",
		JsonRPC: "2.0",
		Params:  []interface{}{},
	}
	payload.Params = append(payload.Params, signature.String())
	payload.Params = append(payload.Params, configStruct{
		Encoding:                       "jsonParsed",
		Commitment:                     "confirmed",
		MaxSupportedTransactionVersion: 0,
	})
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return TransactionResponse{}, err
	}

	req, err := http.NewRequest("POST", env.GetRPCUrl(), bytes.NewBuffer(jsonData))
	if err != nil {
		return TransactionResponse{}, err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TransactionResponse{}, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return TransactionResponse{}, err
	}
	var transactionData TransactionResponse
	err = json.Unmarshal(body, &transactionData)
	if err != nil {
		return TransactionResponse{}, err
	}

	if transactionData.Result == nil {
		return TransactionResponse{}, errors.New("not found")
	}

	return transactionData, nil
}

func GetMultipleConfirmedTransactions(signatures []solana.Signature) ([]TransactionResponse, error) {
	if len(signatures) == 0 {
		return []TransactionResponse{}, nil
	}

	type requestStruct struct {
		ID      string        `json:"id"`
		JsonRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	type encodingStruct struct {
		Encoding   string `json:"encoding"`
		Commitment string `json:"commitment"`
	}

	var payloads []requestStruct
	for _, sig := range signatures {
		payload := requestStruct{
			ID:      string(uuid.NewString()),
			Method:  "getConfirmedTransaction",
			JsonRPC: "2.0",
			Params:  []interface{}{},
		}
		payload.Params = append(payload.Params, sig.String())
		payload.Params = append(payload.Params, encodingStruct{
			Encoding:   "jsonParsed",
			Commitment: "confirmed",
		})

		payloads = append(payloads, payload)
	}

	chunkSize := MaxAccountRequest
	chunks := make([][]requestStruct, (len(payloads)+chunkSize-1)/chunkSize)
	prev := 0
	i := 0
	till := len(payloads) - chunkSize
	for prev < till {
		next := prev + chunkSize
		chunks[i] = payloads[prev:next]
		prev = next
		i++
	}
	chunks[i] = payloads[prev:]

	responses := make([]TransactionResponse, len(signatures))

	var wg sync.WaitGroup
	wg.Add(len(chunks))
	var goerror error

	for i, payload := range chunks {
		go func(p []requestStruct, i int) {
			defer wg.Done()
			j, err := json.Marshal(p)
			if err != nil {
				goerror = err
				return
			}
			req, err := http.NewRequest("POST", env.GetRPCUrl(), bytes.NewBuffer(j))
			if err != nil {
				goerror = err
				return
			}
			req.Header.Set("content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				goerror = err
				return
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				goerror = err
				return
			}
			var transactions []TransactionResponse
			err = json.Unmarshal(body, &transactions)
			if err != nil {
				goerror = err
				return
			}

			if len(transactions) != len(p) {
				goerror = errors.New("invalid RPC response")
				return
			}

			for i2, transaction := range transactions {
				responses[i2+(i*chunkSize)] = transaction
			}
		}(payload, i)
		time.Sleep(getConfirmedTransactionThreadDelay)
	}

	wg.Wait()
	if goerror != nil {
		return []TransactionResponse{}, goerror
	}

	return responses, nil
}
