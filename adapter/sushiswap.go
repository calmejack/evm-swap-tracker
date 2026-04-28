package adapter

import (
	"github.com/calmejack/evm-swap-tracker/config"
)

// NewSushiSwapAdapter returns a DEXAdapter for SushiSwap.
// SushiSwap V1/V2 pairs emit the exact same Swap event signature as
// Uniswap V2, so we reuse UniswapV2Adapter with a different name.
func NewSushiSwapAdapter(cfg *config.Config) *UniswapV2Adapter {
	return NewUniswapV2Adapter(cfg, "SushiSwap")
}
