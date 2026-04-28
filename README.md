# evm-swap-tracker

A Go-based on-chain swap data parsing system that decodes DEX Swap events from
Ethereum, determines the trade direction (**buy** / **sell**), and outputs a
unified `SwapEvent` model.

## Architecture

```
Ethereum Node (RPC / WSS)
        ↓
Block Listener  (subscribes to new blocks)
        ↓
Tx Fetcher      (fetches transaction receipts)
        ↓
Log Parser      (routes logs to the correct DEX adapter)
        ↓
DEX Adapters    (UniswapV2 · UniswapV3 · SushiSwap)
        ↓
SwapEvent output (JSON to stdout)
```

## Supported DEXes

| DEX | Version | Swap Event Signature |
|-----|---------|----------------------|
| Uniswap | V2 | `Swap(address,uint256,uint256,uint256,uint256,address)` |
| Uniswap | V3 | `Swap(address,address,int256,int256,uint160,uint128,int24)` |
| SushiSwap | V1/V2 | same as Uniswap V2 |

## Unified SwapEvent model

```go
type SwapEvent struct {
    TxHash    string  // transaction hash
    Type      string  // "buy" or "sell"
    Dex       string  // "UniswapV2" | "UniswapV3" | "SushiSwap"
    TokenIn   string  // address of token the trader sent
    TokenOut  string  // address of token the trader received
    AmountIn  string  // raw amount (no decimals applied)
    AmountOut string  // raw amount (no decimals applied)
    Sender    string  // initiator address
    Receiver  string  // recipient address
    BlockNum  uint64  // block number
}
```

### Buy / Sell determination

The `Type` field is set by comparing `TokenIn` and `TokenOut` against a
configurable set of *base tokens* (WETH, USDC, USDT, DAI by default):

- **buy**  – the trader spent a base token to acquire another token
- **sell** – the trader sold a token in exchange for a base token

## Configuration

| Environment variable | Default | Description |
|----------------------|---------|-------------|
| `EVM_RPC_URL` | `wss://mainnet.infura.io/ws/v3/YOUR_PROJECT_ID` | WebSocket or HTTPS Ethereum node endpoint |
| `BASE_TOKENS` | WETH, USDC, USDT, DAI (mainnet) | Comma-separated list of quote token addresses |

## Quick start

```bash
# Build
go build -o evm-swap-tracker .

# Run (WebSocket endpoint required for live block subscription)
EVM_RPC_URL=wss://mainnet.infura.io/ws/v3/<key> ./evm-swap-tracker
```

Each decoded swap is printed as indented JSON to stdout:

```json
{
  "tx_hash": "0xabc...",
  "type": "buy",
  "dex": "UniswapV2",
  "token_in": "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
  "token_out": "0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF",
  "amount_in": "1000000000000000000",
  "amount_out": "42000000",
  "sender": "0x...",
  "receiver": "0x...",
  "block_num": 21000000
}
```

## Project layout

```
.
├── main.go              # entry point
├── config/              # runtime configuration
├── models/              # SwapEvent struct
├── adapter/             # DEX-specific log decoders
│   ├── adapter.go       # DEXAdapter interface + shared helpers
│   ├── uniswap_v2.go
│   ├── uniswap_v3.go
│   └── sushiswap.go
├── parser/              # log router
├── fetcher/             # receipt fetcher
└── listener/            # block subscription / polling
```

## Running tests

```bash
go test ./...
```
