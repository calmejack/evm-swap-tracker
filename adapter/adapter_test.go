package adapter_test

import (
	"math/big"
	"testing"

	"github.com/calmejack/evm-swap-tracker/adapter"
	"github.com/calmejack/evm-swap-tracker/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// wellKnown tokens for testing (WETH is a base token in default config)
var (
	weth  = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	usdc  = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	someToken = common.HexToAddress("0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF")
	sender    = common.HexToAddress("0x1111111111111111111111111111111111111111")
	receiver  = common.HexToAddress("0x2222222222222222222222222222222222222222")
)

func defaultCfg() *config.Config {
	return config.Load()
}

// ---------------------------------------------------------------------------
// CanHandle tests
// ---------------------------------------------------------------------------

func TestUniswapV2_CanHandle(t *testing.T) {
	a := adapter.NewUniswapV2Adapter(defaultCfg(), "UniswapV2")

	// valid log: 3 topics starting with the V2 swap topic
	validLog := types.Log{
		Topics: []common.Hash{
			adapter.UniswapV2SwapTopic,
			common.BytesToHash(sender.Bytes()),
			common.BytesToHash(receiver.Bytes()),
		},
	}
	if !a.CanHandle(validLog) {
		t.Error("expected CanHandle=true for V2 Swap log")
	}

	// wrong topic
	wrongLog := types.Log{
		Topics: []common.Hash{
			adapter.UniswapV3SwapTopic, // V3 topic
			common.BytesToHash(sender.Bytes()),
			common.BytesToHash(receiver.Bytes()),
		},
	}
	if a.CanHandle(wrongLog) {
		t.Error("expected CanHandle=false for wrong topic")
	}
}

func TestUniswapV3_CanHandle(t *testing.T) {
	a := adapter.NewUniswapV3Adapter(defaultCfg())

	validLog := types.Log{
		Topics: []common.Hash{
			adapter.UniswapV3SwapTopic,
			common.BytesToHash(sender.Bytes()),
			common.BytesToHash(receiver.Bytes()),
		},
	}
	if !a.CanHandle(validLog) {
		t.Error("expected CanHandle=true for V3 Swap log")
	}
}

func TestSushiSwap_Name(t *testing.T) {
	a := adapter.NewSushiSwapAdapter(defaultCfg())
	if a.Name() != "SushiSwap" {
		t.Errorf("expected Name()=SushiSwap, got %s", a.Name())
	}
}

// ---------------------------------------------------------------------------
// Topic hash correctness
// ---------------------------------------------------------------------------

func TestTopicHashes(t *testing.T) {
	// Verify the pre-computed topic hashes match the expected values from the
	// Ethereum ecosystem.
	expectedV2 := common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822")
	if adapter.UniswapV2SwapTopic != expectedV2 {
		t.Errorf("V2 topic mismatch: got %s want %s", adapter.UniswapV2SwapTopic, expectedV2)
	}

	expectedV3 := common.HexToHash("0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67")
	if adapter.UniswapV3SwapTopic != expectedV3 {
		t.Errorf("V3 topic mismatch: got %s want %s", adapter.UniswapV3SwapTopic, expectedV3)
	}
}

// ---------------------------------------------------------------------------
// swapType logic (via exported helper – tested indirectly through adapters
// by inspecting the "type" field in full ParseSwap unit tests that mock the
// RPC call, but we also verify the configuration level here).
// ---------------------------------------------------------------------------

func TestConfig_BaseTokensDefault(t *testing.T) {
	cfg := defaultCfg()
	if !cfg.BaseTokens[weth] {
		t.Error("WETH should be a base token in default config")
	}
	if !cfg.BaseTokens[usdc] {
		t.Error("USDC should be a base token in default config")
	}
	if cfg.BaseTokens[someToken] {
		t.Error("random token should NOT be a base token")
	}
}

// ---------------------------------------------------------------------------
// Encoding helpers
// ---------------------------------------------------------------------------

// encodeV2Data encodes the four non-indexed uint256 fields for a V2 Swap event.
func encodeV2Data(amount0In, amount1In, amount0Out, amount1Out *big.Int) []byte {
	data := make([]byte, 128)
	amount0In.FillBytes(data[0:32])
	amount1In.FillBytes(data[32:64])
	amount0Out.FillBytes(data[64:96])
	amount1Out.FillBytes(data[96:128])
	return data
}

// encodeV3Data encodes the first two int256 + three trailing uint fields.
// For buy/sell direction tests we only need amount0 and amount1.
func encodeV3Data(amount0, amount1 *big.Int) []byte {
	// ABI-encode two int256 values followed by three zeroed fields
	// (sqrtPriceX96 uint160, liquidity uint128, tick int24).
	data := make([]byte, 5*32)

	// big.Int.FillBytes pads with leading zeros which is what we need for
	// unsigned values; for negative numbers we need two's-complement encoding.
	encode256 := func(dst []byte, n *big.Int) {
		if n.Sign() >= 0 {
			n.FillBytes(dst)
		} else {
			// Two's complement: flip bits and add 1 on a 32-byte word.
			pos := new(big.Int).Neg(n)
			pos.FillBytes(dst)
			// Flip bits.
			for i := range dst {
				dst[i] ^= 0xff
			}
			// Add 1 (propagate carry).
			carry := 1
			for i := 31; i >= 0 && carry > 0; i-- {
				sum := int(dst[i]) + carry
				dst[i] = byte(sum)
				carry = sum >> 8
			}
		}
	}

	encode256(data[0:32], amount0)
	encode256(data[32:64], amount1)
	return data
}
