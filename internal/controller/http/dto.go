package http

import (
	"crypto_pro/internal/domain/entity"
)

type transactions []transaction

type transaction struct {
	Symbol         string  `json:"Symbol"`
	Chain          string  `json:"Chain"`
	MarketFrom     string  `json:"MarketFrom"`
	MarketTo       string  `json:"MarketTo"`
	Spread         float64 `json:"Spread"`
	WithDrawFee    float64 `json:"WithdrawFee"`
	WithdrawMax    float64 `json:"WithdrawMax"`
	AmountCoin     float64 `json:"AmountCoin"`
	AmountAskOrder float64 `json:"AmountAskOrder"`
	AskCost        float64 `json:"AskCost"`
	AskOrder       Orders  `json:"AskOrder"`
	AmountBidOrder float64 `json:"AmountBidOrder"`
	BidCost        float64 `json:"BidCost"`
	BidOrder       Orders  `json:"BidOrder"`
}

type Orders []Order
type Order struct {
	Price float64 `json:"Price"`
	Qty   float64 `json:"Qty"`
}

func (t transactions) toEntity() []entity.Transaction {
	response := []entity.Transaction{}
	for _, val := range t {
		response = append(response, entity.Transaction{
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
			AskOrder:       val.AskOrder.toEntity(),
			AmountBidOrder: val.AmountBidOrder,
			BidCost:        val.BidCost,
			BidOrder:       val.BidOrder.toEntity(),
		})
	}
	return response
}

func (t Orders) toEntity() []entity.Order {
	response := []entity.Order{}
	for _, val := range t {
		response = append(response, entity.Order{
			Price: val.Price,
			Qty:   val.Qty,
		})
	}
	return response
}
