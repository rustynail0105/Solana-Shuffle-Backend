package monitor

import (
	"errors"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	signatureCacheSize = 10
	defaultCommitment  = rpc.CommitmentConfirmed
	defaultDelay       = time.Second
)

type Monitor struct {
	commitment rpc.CommitmentType
	address    solana.PublicKey
	delay      time.Duration

	lastSignatures []solana.Signature

	C chan solana.Signature

	rpcClient *rpc.Client

	stopCh chan struct{}
}

type MonitorConfig struct {
	Commitment rpc.CommitmentType
	Address    solana.PublicKey
	Delay      time.Duration
}

func New(rpcClient *rpc.Client, config MonitorConfig) (*Monitor, error) {
	if config.Address.IsZero() {
		return &Monitor{}, errors.New("address is zero")
	}

	if config.Commitment == "" {
		config.Commitment = defaultCommitment
	}
	if config.Delay == 0 {
		config.Delay = defaultDelay
	}

	c := make(chan solana.Signature, 10)

	monitor := Monitor{
		commitment: config.Commitment,
		address:    config.Address,
		delay:      config.Delay,
		rpcClient:  rpcClient,

		lastSignatures: []solana.Signature{},

		C: c,
	}

	go monitor.routine()

	return &monitor, nil
}

func (m *Monitor) Stop() {
	m.stopCh <- struct{}{}
}
