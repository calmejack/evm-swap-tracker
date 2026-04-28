package adapter

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/calmejack/evm-swap-tracker/config"
	"github.com/calmejack/evm-swap-tracker/models"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// UniswapV3SwapTopic is keccak256("Swap(address,address,int256,int256,uint160,uint128,int24)")
var UniswapV3SwapTopic = crypto.Keccak256Hash([]byte("Swap(address,address,int256,int256,uint160,uint128,int24)"))

// uniswapV3SwapABI describes the non-indexed parameters of the V3 Swap event.
var uniswapV3SwapABI abi.ABI

func init() {
	const rawABI = `[{
		"anonymous": false,
		"inputs": [
			{"indexed": true,  "name": "sender",        "type": "address"},
			{"indexed": true,  "name": "recipient",     "type": "address"},
			{"indexed": false, "name": "amount0",       "type": "int256"},
			{"indexed": false, "name": "amount1",       "type": "int256"},
			{"indexed": false, "name": "sqrtPriceX96",  "type": "uint160"},
			{"indexed": false, "name": "liquidity",     "type": "uint128"},
			{"indexed": false, "name": "tick",          "type": "int24"}
		],
		"name": "Swap",
		"type": "event"
	}]`
	var err error
	uniswapV3SwapABI, err = abi.JSON(strings.NewReader(rawABI))
	if err != nil {
		panic("uniswap_v3: failed to parse swap ABI: " + err.Error())
	}
}

// UniswapV3Adapter handles Swap events emitted by Uniswap V3 pools.
type UniswapV3Adapter struct {
	cfg *config.Config
}

// NewUniswapV3Adapter creates a UniswapV3Adapter.
func NewUniswapV3Adapter(cfg *config.Config) *UniswapV3Adapter {
	return &UniswapV3Adapter{cfg: cfg}
}

func (a *UniswapV3Adapter) Name() string { return "UniswapV3" }

// CanHandle checks for the V3 Swap topic (3 topics: topic0, sender, recipient).
func (a *UniswapV3Adapter) CanHandle(log types.Log) bool {
	return len(log.Topics) == 3 && log.Topics[0] == UniswapV3SwapTopic
}

func (a *UniswapV3Adapter) ParseSwap(
	ctx context.Context,
	log types.Log,
	client *ethclient.Client,
) (*models.SwapEvent, error) {
	// Decode non-indexed fields.
	event, err := uniswapV3SwapABI.Events["Swap"].Inputs.Unpack(log.Data)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3: unpack data: %w", err)
	}

	amount0 := event[0].(*big.Int) // int256: >0 means token0 flows INTO pool
	amount1 := event[1].(*big.Int) // int256: >0 means token1 flows INTO pool

	// Indexed fields.
	sender    := common.BytesToAddress(log.Topics[1].Bytes())
	recipient := common.BytesToAddress(log.Topics[2].Bytes())

	// Fetch pool tokens.
	token0, token1, err := getPoolTokens(ctx, client, log.Address)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v3: get pool tokens: %w", err)
	}

	// In V3 amounts are signed:
	//   amount0 > 0, amount1 < 0 → trader sent token0, received token1
	//   amount1 > 0, amount0 < 0 → trader sent token1, received token0
	var tokenIn, tokenOut common.Address
	var amountIn, amountOut *big.Int

	if amount0.Sign() > 0 {
		tokenIn, amountIn   = token0, amount0
		amountOut            = new(big.Int).Neg(amount1) // make positive
		tokenOut             = token1
	} else {
		tokenIn, amountIn   = token1, amount1
		amountOut            = new(big.Int).Neg(amount0) // make positive
		tokenOut             = token0
	}

	return &models.SwapEvent{
		TxHash:    log.TxHash.Hex(),
		Type:      swapType(tokenIn, tokenOut, a.cfg),
		Dex:       "UniswapV3",
		TokenIn:   tokenIn.Hex(),
		TokenOut:  tokenOut.Hex(),
		AmountIn:  bigString(amountIn),
		AmountOut: bigString(amountOut),
		Sender:    sender.Hex(),
		Receiver:  recipient.Hex(),
		BlockNum:  log.BlockNumber,
	}, nil
}
