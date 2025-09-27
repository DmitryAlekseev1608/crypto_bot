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

func (b TaskUseCase) HandleRequest(requestIn, id string) []entity.Transaction {
	usdt, spreadMin, spreadMax := b.getDataIn(requestIn)
	transactions := b.serverController.GetSpotHandler(usdt, spreadMin, spreadMax)
	for i := range transactions {
		transactions[i].SetID(id)
	}

	if len(transactions) == 0 {
		return nil
	}

	err := b.dbAdapter.UpsertDWHTransactions(transactions)
	if err != nil {
		b.log.Error("Error when upserting transactions: %v", b.log.ErrorC(err))
		return nil
	}

	newTransactions := b.dbAdapter.SelectNewTransactions(id)
	if newTransactions == nil {
		return []entity.Transaction{}
	}

	return newTransactions
}

func (b TaskUseCase) GetAllTransactions(id string) []entity.Transaction {
	transactions := b.dbAdapter.SelectTransactions(id)
	if transactions == nil {
		return []entity.Transaction{}
	}
	return transactions
}

func (b TaskUseCase) getDataIn(input string) (float64, float64, float64) {
	parts := strings.Split(input, " ")
	usdt, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		b.log.Error("Error when transforming usdt '%s': %v", b.log.ErrorC(err), b.log.StringC("input",
			input))
	}
	spreadMin, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		b.log.Error("Error when transforming spreadMin '%s': %v", b.log.ErrorC(err), b.log.StringC("input",
			input))
	}
	spreadMax, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		b.log.Error("Error when transforming spreadMax '%s': %v", b.log.ErrorC(err), b.log.StringC("input",
			input))
	}
	return usdt, spreadMin, spreadMax
}

func (b TaskUseCase) DeleteSession(id string) {
	b.dbAdapter.DeleteSession(id)
}

func (b TaskUseCase) TrancateRawTransactions() {
	b.dbAdapter.TrancateRawTransactions()
}

func (b TaskUseCase) TrancateDwhTransactions() {
	b.dbAdapter.TrancateDwhTransactions()
}

func (b TaskUseCase) GetTransactions(id string) []entity.Transaction {
	return b.dbAdapter.SelectTransactions(id)
}

func (b TaskUseCase) GetInstruction() string {
	return `Привет! Я чат-бот для биржевой аналитики CryptoPro.	Моя основная задача помогать находить наиболее выгодные биржевые транзакции. Я постоянно развиваюсь. На текущий момент я умею работать с биржами:
	- ASCENDEX;
	- BINGX;
	- BITGET;
	- BITMART;
	- BYBIT;
	- HTX;
	- KUKOIN;
	- MEXC;
	- XT.
Просто введи сумму необходимого количества USDT (целое), spread_min, spread_max (до одного знака после запятой) в % через пробел пример 100 0.3 0.5), чтобы я мог искать для тебя транзакции. Для остановки режима сканирования бирж отправь stop в чат, нажми на интересующую сделку и получишь всю необходимую информацию по ней или отправь all, чтобы получить все транзакции сразу.`
}

func (b TaskUseCase) GetInfoAboutTransactions(id string, marketFrom, marketTo, symbol string,
) string {

	transaction := b.dbAdapter.SelectTransactionsBySymbol(id, symbol, marketFrom, marketTo)
	if transaction.ID == "" {
		return "ой, 😀 сделка уже не отслеживается, так как она перестала быть интересной для тебя"
	}
	msgContent := fmt.Sprintf("%v \n", transaction.Symbol)
	msgContent += fmt.Sprintf("📕|%v| \n", transaction.MarketFrom)
	msgContent += fmt.Sprintf("*Сеть:* %v \n", transaction.Chain)
	msgContent += fmt.Sprintf("*Объем б/к:* %.4f %v \n", transaction.AmountCoin, transaction.Symbol)
	msgContent += fmt.Sprintf("*Комиссия:* %v %v \n", transaction.WithDrawFee, transaction.Symbol)
	msgContent += fmt.Sprintf("*Кол-во ордеров:* %v \n", transaction.AmountAskOrder)
	msgContent += fmt.Sprintf("*Стоимость покупки:* %.0f USDT \n", transaction.AskCost)
	msgContent += fmt.Sprintf("*Ордера (Цена/Кол-во):* %v \n", transaction.AskOrder)
	msgContent += fmt.Sprintf("📗|%v| \n", transaction.MarketTo)
	msgContent += fmt.Sprintf("*Кол-во ордеров:* %v \n", transaction.AmountBidOrder)
	msgContent += fmt.Sprintf("*Стоимость продажи:* %.2f USDT \n", transaction.BidCost)
	msgContent += fmt.Sprintf("*Ордера (Цена/Кол-во):* %v \n", transaction.BidOrder)
	msgContent += "--- \n"
	msgContent += fmt.Sprintf("💰 *Спред:* %.2f %%", transaction.Spread)
	return msgContent
}

func (b TaskUseCase) CreateSession(id, requestIn string) {
	usdt, spreadMin, spreadMax := b.getDataIn(requestIn)
	b.dbAdapter.CreateSession(id, usdt, spreadMin, spreadMax)
}
