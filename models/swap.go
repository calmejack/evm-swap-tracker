package models

// SwapEvent is the unified swap model produced by every DEX adapter.
type SwapEvent struct {
	TxHash    string `json:"tx_hash"`
	Type      string `json:"type"`       // "buy" or "sell"
	Dex       string `json:"dex"`        // "UniswapV2" | "UniswapV3" | "SushiSwap"
	TokenIn   string `json:"token_in"`   // address of the token the trader sent
	TokenOut  string `json:"token_out"`  // address of the token the trader received
	AmountIn  string `json:"amount_in"`  // raw amount (no decimals applied)
	AmountOut string `json:"amount_out"` // raw amount (no decimals applied)
	Sender    string `json:"sender"`     // msg.sender / initiator
	Receiver  string `json:"receiver"`   // recipient of tokenOut
	BlockNum  uint64 `json:"block_num"`
}
