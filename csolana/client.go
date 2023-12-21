package csolana

import "github.com/gagliardetto/solana-go/rpc"

type Client struct {
	rpcClient *rpc.Client
	Endpoint  string
}

type ClientConfig struct {
	Endpoint string
}

func NewClient(config ClientConfig) *Client {
	return &Client{
		rpcClient: rpc.New(config.Endpoint),
		Endpoint:  config.Endpoint,
	}
}
