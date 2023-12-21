package vsolana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"net/http"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/google/uuid"
	"github.com/solanashuffle/backend/env"
	"github.com/solanashuffle/backend/utility"
)

const (
	retries = 120
	sleep   = time.Second * 5
)

var (
	wsClient *ws.Client
)

func ConnectWS(wsUrl string) {
	var err error
	wsClient, err = ws.Connect(context.TODO(), wsUrl)
	if err != nil {
		panic(err)
	}
}

func EnsureInstructions(signers []solana.PrivateKey, feePayer solana.PrivateKey, instructions []solana.Instruction) (solana.Signature, error) {
	var signature solana.Signature

	var i int
	for err := errors.New("not done"); err != nil; {
		if i > retries {
			return signature, errors.New("failure to send transaction")
		}
		var i2 int
		for err := errors.New("not done"); err != nil; {
			if i2 > 0 {
				time.Sleep(sleep)
			}
			if i2 > retries {
				return signature, errors.New("failure to send transaction")
			}
			i2++
			signature, err = ExecuteInstructions(signers, feePayer, instructions)
			log.Println(signature, err)
			if err != nil {
				log.Println(signers, feePayer, instructions, "ERR")
			}
		}
		i++
		err = AwaitConfirmedTransaction(signature)
		if err != nil {
			log.Println("ERROR", err)
			time.Sleep(sleep)
		}
	}
	return signature, nil
}

func ExecuteInstructions(signers []solana.PrivateKey, feePayer solana.PrivateKey, instructions []solana.Instruction) (solana.Signature, error) {
	rpcClient := rpc.New(env.GetRPCUrl())

	recent, err := rpcClient.GetRecentBlockhash(context.TODO(), rpc.CommitmentFinalized)
	if err != nil {
		return solana.Signature{}, err
	}

	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(feePayer.PublicKey()),
	)
	if err != nil {
		return solana.Signature{}, err
	}

	_, err = tx.Sign(
		func(key solana.PublicKey) *solana.PrivateKey {
			if feePayer.PublicKey().Equals(key) {
				return &feePayer
			}
			for _, fromAccount := range signers {
				if fromAccount.PublicKey().Equals(key) {
					return &fromAccount
				}
			}
			return nil
		},
	)
	if err != nil {
		return solana.Signature{}, err
	}

	return rpcClient.SendTransaction(
		context.TODO(),
		tx,
	)
}

func AwaitConfirmedTransaction(sig solana.Signature) error {
	if sig == (solana.Signature{}) {
		return nil
	}

	i := 0
	for confirmations := 0; confirmations < 1; {
		if i > 100 {
			return errors.New("transaction not confirmed after 30 seconds")
		}
		if i > 0 {
			time.Sleep(time.Millisecond * 300)
		}
		i++
		resp, err := GetSignatureStatus(sig)
		if err != nil {
			log.Println(err)
			if err.Error() == "transaction failed" {
				return err
			}
			continue
		}
        if rpc.ConfirmationStatusType(resp.Result.Value[0].Confirmationstatus) == rpc.ConfirmationStatusFinalized {
            break
        }
		confirmations = resp.Result.Value[0].Confirmations
		log.Println("confirmations:", confirmations, sig)
	}

	return nil
}

func AwaitFinalizeTransaction(sig solana.Signature) error {
	if sig == (solana.Signature{}) {
		return nil
	}

	i := 0
	for status := ""; status != "finalized"; {
		if i > 240 {
			return errors.New("transaction not finalized after 240 seconds")
		}
		if i != 0 {
			time.Sleep(time.Second)
		}
		i++
		resp, err := GetSignatureStatus(sig)
		if err != nil {
			if err.Error() == "transaction failed" {
				return err
			}
			continue
		}
		log.Println(status)
		status = resp.Result.Value[0].Confirmationstatus
	}
	return nil
}

func AwaitSignatureStatuses(signatures []solana.Signature, status rpc.ConfirmationStatusType) error {
	if len(signatures) <= 0 || signatures[0] == (solana.Signature{}) {
		return nil
	}
	rpcClient := rpc.New(env.GetRPCUrl())

	retries := 100
	if status == rpc.ConfirmationStatusFinalized {
		retries = 800
	}
	sleep := func() {
		time.Sleep(time.Millisecond * 300)
	}
	for i := 0; i < retries; i++ {
		if len(signatures) == 0 {
			return nil
		}
		if i > 0 {
			sleep()
		}
		resp, err := rpcClient.GetSignatureStatuses(
			context.TODO(),
			true,
			signatures...,
		)
		if err != nil {
			log.Println(err)
			continue
		}

		var doneIndeces []int
		for i, resp := range resp.Value {
			if resp == nil {
				continue
			}
			if resp.Err != nil {
				return errors.New("transactions failed")
			}

			if status == rpc.ConfirmationStatusProcessed {
				if resp.ConfirmationStatus == status || resp.ConfirmationStatus == rpc.ConfirmationStatusConfirmed || resp.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
					doneIndeces = append(doneIndeces, i)
				}
			} else if status == rpc.ConfirmationStatusConfirmed {
				if resp.ConfirmationStatus == status || resp.ConfirmationStatus == rpc.ConfirmationStatusFinalized {
					doneIndeces = append(doneIndeces, i)
				}
			} else if status == rpc.ConfirmationStatusFinalized {
				if resp.ConfirmationStatus == status {
					doneIndeces = append(doneIndeces, i)
				}
			}
		}

		for i, s := range doneIndeces {
			signatures = utility.Remove(signatures, s-i)
		}
	}

	return fmt.Errorf("transactions not %s", status)

}

type SignatureResponse struct {
	Jsonrpc string `json:"jsonrpc"`
	Result  struct {
		Context struct {
			Slot int `json:"slot"`
		} `json:"context"`
		Value []struct {
			Confirmationstatus string      `json:"confirmationStatus"`
			Confirmations      int         `json:"confirmations"`
			Err                interface{} `json:"err"`
			Slot               int         `json:"slot"`
			Status             struct {
				Ok interface{} `json:"Ok"`
			} `json:"status"`
		} `json:"value"`
	} `json:"result"`
	ID string `json:"id"`
}

func GetSignatureStatus(signature solana.Signature) (SignatureResponse, error) {
	type requestStruct struct {
		ID      string        `json:"id"`
		JsonRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	type paramStruct struct {
		SearchTransactionHistory bool `json:"searchTransactionHistory"`
	}

	payload := requestStruct{
		ID:      string(uuid.NewString()),
		Method:  "getSignatureStatuses",
		JsonRPC: "2.0",
		Params:  []interface{}{},
	}
	payload.Params = append(payload.Params, []string{signature.String()})
	payload.Params = append(payload.Params, paramStruct{
		SearchTransactionHistory: true,
	})

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return SignatureResponse{}, err
	}

	req, err := http.NewRequest("POST", env.GetRPCUrl(), bytes.NewBuffer(jsonData))
	if err != nil {
		return SignatureResponse{}, err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SignatureResponse{}, err
	}
	if resp.StatusCode != 200 {
		return SignatureResponse{}, errors.New("could not get signature")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return SignatureResponse{}, err
	}
	var signatureData SignatureResponse
	err = json.Unmarshal(body, &signatureData)
	if err != nil {
		return SignatureResponse{}, err
	}
	if len(signatureData.Result.Value) != 1 {
		return SignatureResponse{}, errors.New("invalid response")
	}
	if signatureData.Result.Value == nil {
		return SignatureResponse{}, errors.New("invalid response")
	}
	if signatureData.Result.Value[0].Err != nil {
		return SignatureResponse{}, errors.New("transaction failed")
	}
	return signatureData, nil
}
