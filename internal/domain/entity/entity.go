package entity

import "crypto_pro/internal/configs"

type Chain struct {
	Market      configs.Market
	Symbol      string
	Chain       string
	WithdrawMin float64
	WithdrawFee float64
	WithdrawMax float64
}

type Orders struct {
	Market   configs.Market
	Symbol   string
	AskPrice []Order
	BidPrice []Order
}

type Order struct {
	Price float64
	Qty   float64
}
