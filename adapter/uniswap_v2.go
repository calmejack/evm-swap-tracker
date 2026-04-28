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

// UniswapV2SwapTopic is keccak256("Swap(address,uint256,uint256,uint256,uint256,address)")
var UniswapV2SwapTopic = crypto.Keccak256Hash([]byte("Swap(address,uint256,uint256,uint256,uint256,address)"))

// uniswapV2SwapABI describes the non-indexed parameters of the V2 Swap event.
var uniswapV2SwapABI abi.ABI

func init() {
	const rawABI = `[{
		"anonymous": false,
		"inputs": [
			{"indexed": true,  "name": "sender",     "type": "address"},
			{"indexed": false, "name": "amount0In",  "type": "uint256"},
			{"indexed": false, "name": "amount1In",  "type": "uint256"},
			{"indexed": false, "name": "amount0Out", "type": "uint256"},
			{"indexed": false, "name": "amount1Out", "type": "uint256"},
			{"indexed": true,  "name": "to",         "type": "address"}
		],
		"name": "Swap",
		"type": "event"
	}]`
	var err error
	uniswapV2SwapABI, err = abi.JSON(strings.NewReader(rawABI))
	if err != nil {
		panic("uniswap_v2: failed to parse swap ABI: " + err.Error())
	}
}

// UniswapV2Adapter handles Swap events emitted by Uniswap V2 pairs.
type UniswapV2Adapter struct {
	cfg     *config.Config
	dexName string // "UniswapV2" or "SushiSwap"
}

// NewUniswapV2Adapter creates a UniswapV2Adapter for the given DEX name.
func NewUniswapV2Adapter(cfg *config.Config, dexName string) *UniswapV2Adapter {
	return &UniswapV2Adapter{cfg: cfg, dexName: dexName}
}

func (a *UniswapV2Adapter) Name() string { return a.dexName }

func (a *UniswapV2Adapter) CanHandle(log types.Log) bool {
	return len(log.Topics) == 3 && log.Topics[0] == UniswapV2SwapTopic
}

func (a *UniswapV2Adapter) ParseSwap(
	ctx context.Context,
	log types.Log,
	client *ethclient.Client,
) (*models.SwapEvent, error) {
	// Decode non-indexed fields from Data.
	event, err := uniswapV2SwapABI.Events["Swap"].Inputs.Unpack(log.Data)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v2: unpack data: %w", err)
	}

	amount0In  := event[0].(*big.Int)
	amount1In  := event[1].(*big.Int)
	amount0Out := event[2].(*big.Int)
	amount1Out := event[3].(*big.Int)

	// Indexed fields are stored in Topics[1] and Topics[2].
	sender := common.BytesToAddress(log.Topics[1].Bytes())
	to     := common.BytesToAddress(log.Topics[2].Bytes())

	// Fetch pool tokens.
	token0, token1, err := getPoolTokens(ctx, client, log.Address)
	if err != nil {
		return nil, fmt.Errorf("uniswap_v2: get pool tokens: %w", err)
	}

	// Determine swap direction.
	// amount0In > 0 means the trader sent token0 → received token1.
	// amount1In > 0 means the trader sent token1 → received token0.
	var tokenIn, tokenOut common.Address
	var amountIn, amountOut *big.Int

	if amount0In.Sign() > 0 {
		tokenIn, amountIn   = token0, amount0In
		tokenOut, amountOut = token1, amount1Out
	} else {
		tokenIn, amountIn   = token1, amount1In
		tokenOut, amountOut = token0, amount0Out
	}

	return &models.SwapEvent{
		TxHash:    log.TxHash.Hex(),
		Type:      swapType(tokenIn, tokenOut, a.cfg),
		Dex:       a.dexName,
		TokenIn:   tokenIn.Hex(),
		TokenOut:  tokenOut.Hex(),
		AmountIn:  bigString(amountIn),
		AmountOut: bigString(amountOut),
		Sender:    sender.Hex(),
		Receiver:  to.Hex(),
		BlockNum:  log.BlockNumber,
	}, nil
}
