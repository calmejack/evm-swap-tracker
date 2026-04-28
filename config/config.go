// Package config holds runtime configuration for evm-swap-tracker.
package config

import (
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Config contains all tuneable parameters.
type Config struct {
	// RPCURL is the HTTP(S) or WebSocket endpoint of an Ethereum node.
	// E.g. "wss://mainnet.infura.io/ws/v3/<key>" or "https://rpc.ankr.com/eth"
	RPCURL string

	// BaseTokens is the set of "quote" token addresses (lower-case).
	// A swap where tokenIn is one of these → BUY; tokenOut is one → SELL.
	// Defaults to WETH, USDC, USDT, DAI on Ethereum mainnet.
	BaseTokens map[common.Address]bool
}

// defaultBaseTokens contains well-known Ethereum mainnet quote tokens.
var defaultBaseTokens = []string{
	"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", // WETH
	"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
	"0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
	"0x6B175474E89094C44Da98b954EedeAC495271d0F", // DAI
}

// Load builds a Config from environment variables with sensible defaults.
//
//	EVM_RPC_URL  – Ethereum node endpoint (required for live use)
//	BASE_TOKENS  – comma-separated list of quote token addresses (optional)
func Load() *Config {
	rpcURL := os.Getenv("EVM_RPC_URL")
	if rpcURL == "" {
		rpcURL = "wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID"
	}

	baseTokens := make(map[common.Address]bool)

	raw := os.Getenv("BASE_TOKENS")
	if raw != "" {
		for _, addr := range strings.Split(raw, ",") {
			addr = strings.TrimSpace(addr)
			if addr != "" {
				baseTokens[common.HexToAddress(addr)] = true
			}
		}
	} else {
		for _, addr := range defaultBaseTokens {
			baseTokens[common.HexToAddress(addr)] = true
		}
	}

	return &Config{
		RPCURL:     rpcURL,
		BaseTokens: baseTokens,
	}
}
