package task

import (
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/controller"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/pkg/logger"
	"strconv"
	"strings"
)

var _ usecase.TaskUseCase = (*TaskUseCase)(nil)

type TaskUseCase struct {
	log              logger.Logger
	serverController controller.Server
	dbAdapter        adapters.DbAdapter
}

func New(log logger.Logger) TaskUseCase {
	return TaskUseCase{log: log}
}

func (b TaskUseCase) HandleRequest(requestIn string) []entity.Transaction {
	usdt, spread := b.getDataIn(requestIn)
	transactions := b.serverController.GetSpotHandler(usdt, spread)
	b.dbAdapter.UpsertDWHTransactions(transactions)
	response := make([]entity.Transaction, len(transactions))
	for i, transaction := range transactions {
		response[i] = entity.Transaction{
			Symbol:     transaction.Symbol,
			AmountCoin: transaction.AmountCoin,
			Spread:     transaction.Spread,
			MarketFrom: transaction.MarketFrom,
			MarketTo:   transaction.MarketTo,
			ID:         transaction.ID,
		}
	}
	return response
}

func (b TaskUseCase) getDataIn(input string) (float64, float64) {
	parts := strings.Split(input, " ")
	usdt, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		b.log.Error("Error when transforming '%s': %v", b.log.ErrorC(err), b.log.StringC("input",
			input))
	}
	spread, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		b.log.Error("Error when transforming '%s': %v", b.log.ErrorC(err), b.log.StringC("input",
			input))
	}
	return usdt, spread
}

func (b TaskUseCase) DeleteSession(id int64) {
	b.dbAdapter.DeleteSession(id)
}

func (b TaskUseCase) TrancateRawTransactions() {
	b.dbAdapter.TrancateRawTransactions()
}

func (b TaskUseCase) TrancateDwhTransactions() {
	b.dbAdapter.TrancateDwhTransactions()
}
