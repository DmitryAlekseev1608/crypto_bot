package task

import (
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/controller"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/pkg/logger"
	"fmt"
	"strconv"
	"strings"
)

var _ usecase.TaskUseCase = (*TaskUseCase)(nil)

type TaskUseCase struct {
	log              logger.Logger
	serverController controller.Server
	dbAdapter        adapters.DbAdapter
}

func New(log logger.Logger, serverController controller.Server, dbAdapter adapters.DbAdapter,
) TaskUseCase {
	return TaskUseCase{log: log, serverController: serverController, dbAdapter: dbAdapter}
}

func (b TaskUseCase) HandleRequest(requestIn string, id int64) []entity.Transaction {
	usdt, spread := b.getDataIn(requestIn)
	transactions := b.serverController.GetSpotHandler(usdt, spread)
	for i := range transactions {
		transactions[i].SetID(id)
	}
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

func (b TaskUseCase) GetTransactions(id int64) []entity.Transaction {
	return b.dbAdapter.SelectTransactions(id)
}

func (b TaskUseCase) GetInstruction() string {
	return `
	Привет! Я чат-бот для биржевой аналитики CryptoPro. Моя основная задача помогать находить
	наиболее выгодные биржевые транзакции для спотовых продаж. Я постоянно развиваюсь. На текущий
	момент я умею работать только с биржами BYBIT и KUKOIN. Просто введи сумму необходимого 
	количества USDT (целое) и spread (до одного знака после запятой) в % через пробел
	(пример 100 0.3), чтобы я мог искать для тебя транзакции. Для остановки режима сканирования
	бирж отправь s в чат и смотри последнее полученное сообщение, нажми на интересующую сделку и
	получишь всю необходимую информацию по ней.
	`
}

func (b TaskUseCase) GetInfoAboutTransactions(id int64, marketFrom, marketTo, symbol string,
) string {

	transaction := b.dbAdapter.SelectTransactionsBySymbol(id, symbol, marketFrom, marketTo)
	if transaction.ID == 0 {
		return "ой, 😀 сделка уже не отслеживается, так как она перестала быть интересной для тебя"
	}
	msgContent := fmt.Sprintf("%v \n", transaction.Symbol)
	msgContent += fmt.Sprintf("📕|%v| \n", transaction.MarketFrom)
	msgContent += fmt.Sprintf("Комиссия: %v %v \n", transaction.WithDrawFee, transaction.Symbol)
	msgContent += fmt.Sprintf("Объем допустимый: %v %v \n", transaction.WithdrawMax,
		transaction.Symbol)
	msgContent += fmt.Sprintf("Сеть: %v \n", transaction.Chain)
	msgContent += fmt.Sprintf("Объем: %.4f %v \n", transaction.AmountCoin, transaction.Symbol)
	msgContent += fmt.Sprintf("Кол-во ордеров: %v \n", transaction.AmountAskOrder)
	msgContent += fmt.Sprintf("Стоимость: %.2f USDT \n", transaction.AskCost)
	msgContent += fmt.Sprintf("Ордера (Цена/Кол-во): %v \n", transaction.AskOrder)
	msgContent += fmt.Sprintf("📗|%v| \n", transaction.MarketTo)
	msgContent += fmt.Sprintf("Кол-во ордеров: %v \n", transaction.AmountBidOrder)
	msgContent += fmt.Sprintf("Стоимость: %.2f USDT \n", transaction.BidCost)
	msgContent += fmt.Sprintf("Ордера (Цена/Кол-во): %v \n", transaction.BidOrder)
	msgContent += "--- \n"
	msgContent += fmt.Sprintf("💰 Спред: %.2f %%", transaction.Spread)
	return msgContent
}
