package monitor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

func (m *Monitor) routine() {
	ticker := time.NewTicker(m.delay)

	m.task()

	for {
		select {
		case <-ticker.C:
			signatures, err := m.task()
			if err != nil {
				log.Println(err)
				continue
			}

			if len(signatures) == 0 {
				continue
			}

			// found new signatures
			for _, signature := range signatures {
				m.C <- signature
			}
		case <-m.stopCh:
			return
		}
	}
}

func (m *Monitor) task() ([]solana.Signature, error) {
	log.Println("running task")

	opts := &rpc.GetSignaturesForAddressOpts{
		Commitment: m.commitment,
	}

	if len(m.lastSignatures) > 0 {
		opts.Until = m.lastSignatures[0]
	}

	t1 := time.Now()
	resp, err := m.rpcClient.GetSignaturesForAddressWithOpts(
		context.TODO(),
		m.address,
		opts,
	)
	if err != nil {
		return []solana.Signature{}, err
	}
	fmt.Println(time.Since(t1))

	for _, txSignature := range resp {
		for _, lastSignature := range m.lastSignatures {
			if txSignature.Signature == lastSignature {
				return []solana.Signature{}, errors.New("found old signature")
			}
		}
	}

	var signatures []solana.Signature
	for _, txSignature := range resp {
		if txSignature.Err != nil {
			continue
		}

		signatures = append(signatures, txSignature.Signature)
	}

	m.lastSignatures = append(signatures, m.lastSignatures...)

	if len(m.lastSignatures) > signatureCacheSize {
		m.lastSignatures = m.lastSignatures[:signatureCacheSize]
	}

	return signatures, nil
}
