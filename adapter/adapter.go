// Package adapter defines the interface that every DEX adapter must satisfy,
// plus shared helpers used by multiple adapters.
package adapter

import (
	"context"
	"math/big"
	"strings"

	"github.com/calmejack/evm-swap-tracker/config"
	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DEXAdapter parses Swap logs emitted by a specific DEX family.
type DEXAdapter interface {
	// Name returns the human-readable DEX name, e.g. "UniswapV2".
	Name() string

	// CanHandle returns true when this adapter recognises the given log.
	CanHandle(log types.Log) bool

	// ParseSwap converts a raw Ethereum log into a SwapEvent.
	ParseSwap(ctx context.Context, log types.Log, client *ethclient.Client) (*models.SwapEvent, error)
}

// swapType returns "buy" when tokenIn is a base/quote token (WETH, USDC …),
// and "sell" when tokenOut is a base/quote token.
// If neither side matches the heuristic falls back to "buy".
func swapType(tokenIn, tokenOut common.Address, cfg *config.Config) string {
	if cfg.BaseTokens[tokenIn] {
		return "buy"
	}
	if cfg.BaseTokens[tokenOut] {
		return "sell"
	}
	return "buy"
}

// minimalPairABI is just enough to call token0() and token1() on any
// Uniswap-style pair / pool contract.
var minimalPairABI abi.ABI

func init() {
	const rawABI = `[
		{"inputs":[],"name":"token0","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},
		{"inputs":[],"name":"token1","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}
	]`
	var err error
	minimalPairABI, err = abi.JSON(strings.NewReader(rawABI))
	if err != nil {
		panic("adapter: failed to parse minimal pair ABI: " + err.Error())
	}
}

// getPoolTokens returns the token0 and token1 addresses for a Uniswap-style
// pool by making two eth_call requests.
func getPoolTokens(ctx context.Context, client *ethclient.Client, pool common.Address) (token0, token1 common.Address, err error) {
	t0Data, err := minimalPairABI.Pack("token0")
	if err != nil {
		return token0, token1, err
	}
	t1Data, err := minimalPairABI.Pack("token1")
	if err != nil {
		return token0, token1, err
	}

	res0, err := client.CallContract(ctx, ethereum.CallMsg{To: &pool, Data: t0Data}, nil)
	if err != nil {
		return token0, token1, err
	}
	res1, err := client.CallContract(ctx, ethereum.CallMsg{To: &pool, Data: t1Data}, nil)
	if err != nil {
		return token0, token1, err
	}

	var out0, out1 []interface{}
	if out0, err = minimalPairABI.Unpack("token0", res0); err != nil {
		return token0, token1, err
	}
	if out1, err = minimalPairABI.Unpack("token1", res1); err != nil {
		return token0, token1, err
	}

	token0 = out0[0].(common.Address)
	token1 = out1[0].(common.Address)
	return token0, token1, nil
}

// bigString converts a *big.Int to its decimal string representation,
// returning "0" when the pointer is nil.
func bigString(n *big.Int) string {
	if n == nil {
		return "0"
	}
	return n.String()
}
