package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/google/uuid"
)

const (
	retries = 120
	sleep   = time.Second * 5
)

var (
	bot1 = solana.MustPrivateKeyFromBase58("4qd97A9WHBzb96jiTZMJzgyg3uWCocctED98E7xoWw6WHjm3HzNADdHAjkvnviZ2my2WgirjDkixgrADi2bYhKC5")
	bot2 = solana.MustPrivateKeyFromBase58("3i7qai3oKQYjB4MUzoa2XQrdeYstMYr9tFH3rUtapRmH8tc9ecGynBrPmR2w2f2hVjAzevArnhXBAiH8vUA9tQyJ")
	bot3 = solana.MustPrivateKeyFromBase58("42dE3Nso4gqSGg4i6JdiY3JYme2FWrRhUFCFqQcQQWv3EB5hQqX1JqQ7W2xK3JyWR5D5cgvnrAqo2LcLGruJ2UxE")

	amount = 100_000

	rpc1 = "https://lively-cool-hill.solana-mainnet.quiknode.pro/c9cdc92c17469a3cc71f79fbbdbf9f6fa6d973e8/"
	rpc2 = "https://solanashuffle.nodedoctor.io/"
	rpc3 = "https://immortals.deeznode.io/token=1672944359-eO8zOw5CpMxwKt5XIyBn4bXFfHmyGfP8rCAJbqGGaCk%3D"
)

func main() {
	var rpc1Signatures []solana.Signature
	var rpc2Signatures []solana.Signature
	var rpc3Signatures []solana.Signature

	var rpc1Times []time.Duration
	var rpc2Times []time.Duration
	var rpc3Times []time.Duration

	iterations := 10

	for i := 0; i < iterations; i++ {
		var wg sync.WaitGroup
		wg.Add(3)
		go func(wg *sync.WaitGroup) {
			start := time.Now()
			fmt.Println("running quick", start)
			defer wg.Done()
			signature, err := benchmarkTX(rpc1, bot1, bot2.PublicKey())
			if err != nil {
				panic(err)
			}
			rpc1Signatures = append(rpc1Signatures, signature)
			since := time.Since(start)
			rpc1Times = append(rpc1Times, since)
			fmt.Println(since, "quick")
		}(&wg)

		go func(wg *sync.WaitGroup) {
			start := time.Now()
			fmt.Println("running doctor", start)
			defer wg.Done()
			signature, err := benchmarkTX(rpc2, bot2, bot1.PublicKey())
			if err != nil {
				panic(err)
			}
			rpc2Signatures = append(rpc2Signatures, signature)
			since := time.Since(start)
			rpc2Times = append(rpc2Times, since)
			fmt.Println(since, "doctor")
		}(&wg)

		go func(wg *sync.WaitGroup) {
			start := time.Now()
			fmt.Println("running deeze", start)
			defer wg.Done()
			signature, err := benchmarkTX(rpc3, bot3, bot2.PublicKey())
			if err != nil {
				panic(err)
			}
			rpc3Signatures = append(rpc3Signatures, signature)
			since := time.Since(start)
			rpc3Times = append(rpc3Times, since)
			fmt.Println(since, "deeze")
		}(&wg)

		wg.Wait()
	}

	var rpc1Highest time.Duration
	var rpc1Average time.Duration
	for _, time := range rpc1Times {
		rpc1Average += time
		if time > rpc1Highest {
			rpc1Highest = time
		}
	}

	var rpc2Highest time.Duration
	var rpc2Average time.Duration
	for _, time := range rpc2Times {
		rpc2Average += time
		if time > rpc2Highest {
			rpc2Highest = time
		}
	}

	var rpc3Highest time.Duration
	var rpc3Average time.Duration
	for _, time := range rpc3Times {
		rpc3Average += time
		if time > rpc3Highest {
			rpc3Highest = time
		}
	}

	rpc1Avg := int(rpc1Average) / (len(rpc1Times))
	rpc2Avg := int(rpc2Average) / (len(rpc2Times))
	rpc3Avg := int(rpc3Average) / (len(rpc3Times))

	fmt.Println("---------------------")
	fmt.Println(rpc1Avg, rpc1Highest, "quick")
	fmt.Println(rpc2Avg, rpc2Highest, "doctor")
	fmt.Println(rpc3Avg, rpc2Highest, "deeze")
	fmt.Println("factor", float64(rpc1Avg)/float64(rpc2Avg))
}

func benchmarkTX(rpcUrl string, sender solana.PrivateKey, receiver solana.PublicKey) (solana.Signature, error) {
	instructions := SendSOLInstructions(sender, receiver, amount)
	return EnsureInstructions(rpcUrl, sender, sender, instructions)
}

func EnsureInstructions(rpcUrl string, fromAccount solana.PrivateKey, feePayer solana.PrivateKey, instructions []solana.Instruction) (solana.Signature, error) {
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
			signature, err = ExecuteInstructions(rpcUrl, fromAccount, feePayer, instructions)
			fmt.Println(signature, err)
			if err != nil {
				fmt.Println(fromAccount, feePayer, instructions, "ERR")
				fmt.Println(fromAccount.PublicKey())
			}
		}
		i++
		err = AwaitConfirmedTransaction("https://lively-cool-hill.solana-mainnet.quiknode.pro/c9cdc92c17469a3cc71f79fbbdbf9f6fa6d973e8/", signature)
		if err != nil {
			fmt.Println("ERROR", err)
			time.Sleep(sleep)
		}
	}
	return signature, nil
}

func ExecuteInstructions(rpcUrl string, fromAccount solana.PrivateKey, feePayer solana.PrivateKey, instructions []solana.Instruction) (solana.Signature, error) {
	rpcClient := rpc.New(rpcUrl)

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
			if fromAccount.PublicKey().Equals(key) {
				return &fromAccount
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

func AwaitConfirmedTransaction(rpc string, sig solana.Signature) error {
	if sig == (solana.Signature{}) {
		return nil
	}

	i := 0
	for confirmations := 0; confirmations < 1; {
		if i > 300 {
			return errors.New("transaction not confirmed after 30 seconds")
		}
		if i > 0 {
			time.Sleep(time.Millisecond * 100)
		}
		i++
		resp, err := GetSignatureStatus(rpc, sig)
		if err != nil {
			fmt.Println(err)
			if err.Error() == "transaction failed" {
				return err
			}
			continue
		}
		confirmations = resp.Result.Value[0].Confirmations
		fmt.Println("confirmations:", confirmations, sig)
	}

	return nil
}

func AwaitConfirmedTransaction2(wsClient *ws.Client, sig solana.Signature) error {
	sub, err := wsClient.SignatureSubscribe(
		sig,
		rpc.CommitmentConfirmed,
	)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	timeout := 2 * time.Minute // random default timeout

	for {
		select {
		case <-time.After(timeout):
			return errors.New("timout error")
		case resp, ok := <-sub.Response():
			if !ok {
				return fmt.Errorf("subscription closed")
			}
			if resp.Value.Err != nil {
				// The transaction was confirmed, but it failed while executing (one of the instructions failed).
				return fmt.Errorf("confirmed transaction with execution error: %v", resp.Value.Err)
			} else {
				// Success! Confirmed! And there was no error while executing the transaction.
				return nil
			}
		case err := <-sub.Err():
			return err
		}
	}
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

func GetSignatureStatus(rpc string, signature solana.Signature) (SignatureResponse, error) {
	type requestStruct struct {
		ID      string        `json:"id"`
		JsonRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
	}
	type paramStruct struct {
		SearchTransactionHistory bool `json:"searchTransactionHistory"`
	}

	u := uuid.NewString()

	payload := requestStruct{
		ID:      string(u),
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

	req, err := http.NewRequest("POST", rpc, bytes.NewBuffer(jsonData))
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

func SendSOLInstructions(fromAccount solana.PrivateKey, toAccount solana.PublicKey, lamports int) []solana.Instruction {
	return []solana.Instruction{
		system.NewTransferInstruction(
			uint64(lamports),
			fromAccount.PublicKey(),
			toAccount,
		).Build(),
	}
}
