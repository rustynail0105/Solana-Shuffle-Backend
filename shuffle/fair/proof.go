package fair

import "github.com/gagliardetto/solana-go"

type Proof struct {
	BlockHash     solana.Hash      `json:"blockHash" bson:"blockHash"`
	BlockSlot     uint64           `json:"blockSlot" bson:"blockSlot"`
	FairSignature solana.Signature `json:"fairSignature" bson:"fairSignature"`
}
