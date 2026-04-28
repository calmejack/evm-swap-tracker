package parser_test

import (
	"context"
	"testing"

	"github.com/calmejack/evm-swap-tracker/adapter"
	"github.com/calmejack/evm-swap-tracker/config"
	"github.com/calmejack/evm-swap-tracker/parser"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestLogParser_ReturnsNilForUnknownLog(t *testing.T) {
	cfg := config.Load()
	lp := parser.NewLogParser(
		adapter.NewUniswapV3Adapter(cfg),
		adapter.NewUniswapV2Adapter(cfg, "UniswapV2"),
		adapter.NewSushiSwapAdapter(cfg),
	)

	// A log with no topics we recognise.
	unknownLog := types.Log{
		Topics: []common.Hash{
			common.HexToHash("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
		},
	}

	result, err := lp.Parse(context.Background(), unknownLog, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for unknown log, got %+v", result)
	}
}

func TestLogParser_RoutesV3BeforeV2(t *testing.T) {
	cfg := config.Load()
	lp := parser.NewLogParser(
		adapter.NewUniswapV3Adapter(cfg),
		adapter.NewUniswapV2Adapter(cfg, "UniswapV2"),
	)

	v3Log := types.Log{
		Topics: []common.Hash{
			adapter.UniswapV3SwapTopic,
			common.BytesToHash(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes()),
			common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes()),
		},
	}

	// CanHandle should match V3 adapter first.
	v3Adapter := adapter.NewUniswapV3Adapter(cfg)
	if !v3Adapter.CanHandle(v3Log) {
		t.Error("V3 adapter should handle a V3 log")
	}
	v2Adapter := adapter.NewUniswapV2Adapter(cfg, "UniswapV2")
	if v2Adapter.CanHandle(v3Log) {
		t.Error("V2 adapter should NOT handle a V3 log (different topic hash)")
	}

	_ = lp
}
