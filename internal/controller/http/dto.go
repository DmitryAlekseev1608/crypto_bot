package http

import (
	"crypto_pro/internal/domain/entity"
)

type transactions []transaction

type transaction struct {
	ID             int     `json:"id"`
	Symbol         string  `json:"symbol"`
	Chain          string  `json:"chain"`
	MarketFrom     string  `json:"market_from"`
	MarketTo       string  `json:"market_to"`
	Spread         float64 `json:"spread"`
	WithDrawFee    float64 `json:"with_draw_fee"`
	WithdrawMax    float64 `json:"withdraw_max"`
	AmountCoin     float64 `json:"amount_coin"`
	AmountAskOrder float64 `json:"amount_ask_order"`
	AskCost        float64 `json:"ask_cost"`
	AskOrder       float64 `json:"ask_order"`
	AmountBidOrder float64 `json:"amount_bid_order"`
	BidCost        float64 `json:"bid_cost"`
	BidOrder       float64 `json:"bid_order"`
}

func (t transactions) toEntity() []entity.Transaction {
	response := []entity.Transaction{}
	for _, val := range t {
		response = append(response, entity.Transaction{
			ID:             val.ID,
			Symbol:         val.Symbol,
			Chain:          val.Chain,
			MarketFrom:     val.MarketFrom,
			MarketTo:       val.MarketTo,
			Spread:         val.Spread,
			WithDrawFee:    val.WithDrawFee,
			WithdrawMax:    val.WithdrawMax,
			AmountCoin:     val.AmountCoin,
			AmountAskOrder: val.AmountAskOrder,
			AskCost:        val.AskCost,
			AskOrder:       val.AskOrder,
			AmountBidOrder: val.AmountBidOrder,
			BidCost:        val.BidCost,
			BidOrder:       val.BidOrder,
		})
	}
	return response
}
