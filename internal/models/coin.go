package models

const (
	Bitcoin  = "BTC"
	Ethereum = "ETH"
	Atom     = "ATOM"
	Polkadot = "DOT"
	Chia     = "XCH"
)

type Coin struct {
	Ticker string
	Price  float64
}
