package postgres

import (
	"crypto_pro/internal/domain/entity"
	"encoding/json"
	"time"
)

type transactions []transaction

type transaction struct {
	ID             string          `db:"id"`
	Symbol         string          `db:"symbol"`
	Chain          string          `db:"chain"`
	MarketFrom     string          `db:"market_from"`
	MarketTo       string          `db:"market_to"`
	Spread         float64         `db:"spread"`
	WithDrawFee    float64         `db:"with_draw_fee"`
	WithdrawMax    float64         `db:"withdraw_max"`
	AmountCoin     float64         `db:"amount_coin"`
	AmountAskOrder float64         `db:"amount_ask_order"`
	AskCost        float64         `db:"ask_cost"`
	AskOrder       json.RawMessage `db:"ask_order"`
	AmountBidOrder float64         `db:"amount_bid_order"`
	BidCost        float64         `db:"bid_cost"`
	BidOrder       json.RawMessage `db:"bid_order"`
	IsPosted       bool            `db:"is_posted"`
	UpdatedAt      time.Time       `db:"updated_at"`
}

func (t transactions) toEntity() []entity.Transaction {
	response := []entity.Transaction{}
	for _, val := range t {

		askOrder := []entity.Order{}
		if json.Unmarshal(val.AskOrder, &askOrder) != nil {
			return nil
		}

		bidOrder := []entity.Order{}
		if json.Unmarshal(val.BidOrder, &bidOrder) != nil {
			return nil
		}

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
			AskOrder:       askOrder,
			AmountBidOrder: val.AmountBidOrder,
			BidCost:        val.BidCost,
			BidOrder:       bidOrder,
		})
	}
	return response
}

func fromEntityToModel(transactionsEntity []entity.Transaction) (transactions, error) {

	transactions := transactions{}

	for _, transactionRow := range transactionsEntity {
		askOrderJSON, err := json.Marshal(transactionRow.AskOrder)
		if err != nil {
			return nil, err
		}
		bidOrderJSON, err := json.Marshal(transactionRow.BidOrder)
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, transaction{
			ID:             transactionRow.ID,
			Symbol:         transactionRow.Symbol,
			Chain:          transactionRow.Chain,
			MarketFrom:     transactionRow.MarketFrom,
			MarketTo:       transactionRow.MarketTo,
			Spread:         transactionRow.Spread,
			WithDrawFee:    transactionRow.WithDrawFee,
			WithdrawMax:    transactionRow.WithdrawMax,
			AmountCoin:     transactionRow.AmountCoin,
			AmountAskOrder: transactionRow.AmountAskOrder,
			AskCost:        transactionRow.AskCost,
			AskOrder:       askOrderJSON,
			AmountBidOrder: transactionRow.AmountBidOrder,
			BidCost:        transactionRow.BidCost,
			BidOrder:       bidOrderJSON,
		})
	}

	return transactions, nil
}
