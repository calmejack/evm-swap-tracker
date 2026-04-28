// Package fetcher retrieves transaction receipts from an Ethereum node.
package fetcher

import (
	"context"
	"fmt"

	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/calmejack/evm-swap-tracker/parser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func ethHashFromString(s string) common.Hash {
	return common.HexToHash(s)
}

// TxFetcher fetches receipts and extracts swap events from their logs.
type TxFetcher struct {
	client *ethclient.Client
	parser *parser.LogParser
}

// NewTxFetcher constructs a TxFetcher.
func NewTxFetcher(client *ethclient.Client, parser *parser.LogParser) *TxFetcher {
	return &TxFetcher{client: client, parser: parser}
}

// ProcessReceipt iterates over all logs in a receipt and returns the
// decoded SwapEvents (skipping logs that are not recognised swap events).
func (f *TxFetcher) ProcessReceipt(ctx context.Context, receipt *types.Receipt) ([]*models.SwapEvent, error) {
	var swaps []*models.SwapEvent
	for _, log := range receipt.Logs {
		if log == nil {
			continue
		}
		swap, err := f.parser.Parse(ctx, *log, f.client)
		if err != nil {
			return nil, fmt.Errorf("fetcher: tx %s log %d: %w", receipt.TxHash.Hex(), log.Index, err)
		}
		if swap != nil {
			swaps = append(swaps, swap)
		}
	}
	return swaps, nil
}

// FetchAndProcess retrieves the receipt for txHash and returns all swap events.
func (f *TxFetcher) FetchAndProcess(ctx context.Context, txHash string) ([]*models.SwapEvent, error) {
	hash := ethHashFromString(txHash)
	receipt, err := f.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("fetcher: get receipt %s: %w", txHash, err)
	}
	return f.ProcessReceipt(ctx, receipt)
}
