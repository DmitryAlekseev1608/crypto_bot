package entity

import "time"

type Transaction struct {
	ID             string
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
	AskOrder       []Order
	AmountBidOrder float64
	BidCost        float64
	BidOrder       []Order
	IsPosted       bool
	UpdatedAt      time.Time
}

type Order struct {
	Price float64
	Qty   float64
}

func (t *Transaction) SetID(id string) {
	t.ID = id
}

type Session struct {
	ID     string
	USDT   float64
	Spread float64
}
