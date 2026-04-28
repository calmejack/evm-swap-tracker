// Package parser routes raw Ethereum logs to the correct DEX adapter.
package parser

import (
	"context"
	"fmt"

	"github.com/calmejack/evm-swap-tracker/adapter"
	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// LogParser dispatches Ethereum logs to registered DEX adapters.
type LogParser struct {
	adapters []adapter.DEXAdapter
}

// NewLogParser creates a LogParser with the provided adapters.
func NewLogParser(adapters ...adapter.DEXAdapter) *LogParser {
	return &LogParser{adapters: adapters}
}

// Parse attempts to decode a swap event from the given log.
// It iterates through all registered adapters and returns the first
// successful result. Returns (nil, nil) when no adapter can handle the log.
func (p *LogParser) Parse(ctx context.Context, log types.Log, client *ethclient.Client) (*models.SwapEvent, error) {
	for _, a := range p.adapters {
		if !a.CanHandle(log) {
			continue
		}
		swap, err := a.ParseSwap(ctx, log, client)
		if err != nil {
			return nil, fmt.Errorf("parser: %s: %w", a.Name(), err)
		}
		return swap, nil
	}
	return nil, nil // not a recognised swap log
}
