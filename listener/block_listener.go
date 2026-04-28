// Package listener subscribes to new Ethereum blocks and feeds each
// transaction through the swap-parsing pipeline.
package listener

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/calmejack/evm-swap-tracker/fetcher"
	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// SwapHandler is called for every swap event decoded from a new block.
type SwapHandler func(swap *models.SwapEvent)

// BlockListener subscribes to new block headers and processes each
// transaction receipt through the TxFetcher / LogParser pipeline.
type BlockListener struct {
	client  *ethclient.Client
	fetcher *fetcher.TxFetcher
	handler SwapHandler
}

// NewBlockListener constructs a BlockListener.
func NewBlockListener(client *ethclient.Client, f *fetcher.TxFetcher, handler SwapHandler) *BlockListener {
	return &BlockListener{client: client, fetcher: f, handler: handler}
}

// Listen subscribes to new block headers on the node.
// It blocks until ctx is cancelled or a fatal error occurs.
// NOTE: requires a WebSocket-capable endpoint (wss://).
func (l *BlockListener) Listen(ctx context.Context) error {
	headers := make(chan *types.Header)
	sub, err := l.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return fmt.Errorf("listener: subscribe new head: %w", err)
	}
	defer sub.Unsubscribe()

	log.Println("listener: subscribed to new block headers")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-sub.Err():
			return fmt.Errorf("listener: subscription error: %w", err)

		case header := <-headers:
			go l.processBlock(ctx, header.Number)
		}
	}
}

// PollBlocks is an alternative to Listen for HTTP (non-WS) endpoints.
// It polls the latest block number and processes any blocks it hasn't
// seen yet. Call it in a loop with a suitable sleep interval.
func (l *BlockListener) PollBlocks(ctx context.Context, from *big.Int) (*big.Int, error) {
	latest, err := l.client.BlockNumber(ctx)
	if err != nil {
		return from, fmt.Errorf("listener: get latest block: %w", err)
	}
	latestBig := new(big.Int).SetUint64(latest)

	for n := new(big.Int).Set(from); n.Cmp(latestBig) <= 0; n.Add(n, big.NewInt(1)) {
		l.processBlock(ctx, new(big.Int).Set(n))
	}
	return new(big.Int).Add(latestBig, big.NewInt(1)), nil
}

// processBlock fetches the full block, iterates over its transactions,
// retrieves their receipts, and calls the SwapHandler for each event.
func (l *BlockListener) processBlock(ctx context.Context, blockNum *big.Int) {
	block, err := l.client.BlockByNumber(ctx, blockNum)
	if err != nil {
		log.Printf("listener: block %s: get block: %v", blockNum, err)
		return
	}

	for _, tx := range block.Transactions() {
		// Skip transactions that are not calls (e.g. plain ETH transfers).
		if tx.To() == nil {
			continue
		}

		receipt, err := l.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			log.Printf("listener: tx %s: get receipt: %v", tx.Hash().Hex(), err)
			continue
		}
		if receipt.Status != 1 {
			continue // failed transaction
		}

		swaps, err := l.fetcher.ProcessReceipt(ctx, receipt)
		if err != nil {
			log.Printf("listener: tx %s: process receipt: %v", tx.Hash().Hex(), err)
			continue
		}

		for _, swap := range swaps {
			l.handler(swap)
		}
	}

	// Suppress "ethereum: not found" when no matching logs exist.
	_ = ethereum.NotFound
}
