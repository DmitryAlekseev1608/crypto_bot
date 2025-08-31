package entity

import "time"

type Transaction struct {
	ID             int
	Symbol         string
	Chain          string
	MarketFrom     string
	MarketTo       string
	Spread         float64
	WithDrawFee    float64
	WithdrawMax    float64
	AmountCoin     float64
	AmountAskOrder float64
	AskCost        float64
	AskOrder       float64
	AmountBidOrder float64
	BidCost        float64
	BidOrder       float64
	UpdatedAt      time.Time
}
