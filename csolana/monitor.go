package csolana

import (
	"github.com/solanashuffle/backend/csolana/monitor"
)

func (c *Client) NewMonitor(config monitor.MonitorConfig) (*monitor.Monitor, error) {
	return monitor.New(c.rpcClient, config)
}
