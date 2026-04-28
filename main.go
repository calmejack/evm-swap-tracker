// evm-swap-tracker is a chain-data parsing system that decodes on-chain
// Swap events from Uniswap V2, Uniswap V3, and SushiSwap, determines the
// trade direction (buy / sell), and outputs a unified SwapEvent model.
//
// Usage:
//
//	EVM_RPC_URL=wss://mainnet.infura.io/ws/v3/<key> go run .
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/calmejack/evm-swap-tracker/adapter"
	"github.com/calmejack/evm-swap-tracker/config"
	"github.com/calmejack/evm-swap-tracker/fetcher"
	"github.com/calmejack/evm-swap-tracker/listener"
	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/calmejack/evm-swap-tracker/parser"
	"github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	cfg := config.Load()

	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		log.Fatalf("main: dial %s: %v", cfg.RPCURL, err)
	}
	defer client.Close()

	// Build the adapter chain: V3 must be checked before V2/SushiSwap because
	// both V2 and V3 produce a "Swap" event but with different signatures.
	adapters := []adapter.DEXAdapter{
		adapter.NewUniswapV3Adapter(cfg),
		adapter.NewUniswapV2Adapter(cfg, "UniswapV2"),
		adapter.NewSushiSwapAdapter(cfg),
	}

	lp := parser.NewLogParser(adapters...)
	tf := fetcher.NewTxFetcher(client, lp)

	handler := func(swap *models.SwapEvent) {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(swap); err != nil {
			log.Printf("main: encode swap: %v", err)
		}
	}

	bl := listener.NewBlockListener(client, tf, handler)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("main: starting block listener on %s", cfg.RPCURL)
	if err := bl.Listen(ctx); err != nil && err != context.Canceled {
		log.Fatalf("main: listener exited: %v", err)
	}
	log.Println("main: shutdown complete")
}
