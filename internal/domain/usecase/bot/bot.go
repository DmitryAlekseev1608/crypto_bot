package bot

import "crypto_pro/pkg/logger"

type BotUseCase struct {
	log logger.Logger
}

func New(log logger.Logger) BotUseCase {
	return BotUseCase{log: log}
}

func (b BotUseCase) Run() {
	b.log.Info("Bot started")
}

func (b BotUseCase) HandleMessage(text string) string {
	re := regexp.MustCompile(`^\d+\s\d+(\.\d+)?$`)

	switch {
	case text == "Инструкция":
		return b.getInstruction()
	case re.MatchString(text):
		if _, exists := activeSessions[update.Message.Chat.ID]; exists {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сессия активна"))
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		activeSessions[update.Message.Chat.ID] = cancel
		go processRequest(ctx, bot, update.Message)
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сессия начата. Отправьте 's' для отмены."))
	case text == "s":
		if cancel, exists := activeSessions[update.Message.Chat.ID]; exists {
			cancel()
			delete(activeSessions, update.Message.Chat.ID)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сессия отменена."))
		} else {
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Нет активной сессии."))
		}
	default:
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Такого действия ботом не предусмотрено или что-то было введено не верно")
		msg.ReplyMarkup = keyboard
		bot.Send(msg)
	}
} else if update.CallbackQuery != nil {
	callbackQuery := update.CallbackQuery
	logger.Infof("User pressed: %v", callbackQuery.Data)
	dataFromRedis := redis.GetOrder(1, callbackQuery.Data)
	var dataForClient redis.ClientInfoVal
	err := json.Unmarshal([]byte(dataFromRedis), &dataForClient)
	if err != nil {
		logger.Warnf("Error for unmarshal JSON: %v", err)
	}
	if dataForClient.WithDrawFee != 0 {
		textWithDeal := fmt.Sprintf("%v \n", dataForClient.Symbol)
		textWithDeal += fmt.Sprintf("📕|%v| \n", dataForClient.MarketFrom)
		textWithDeal += fmt.Sprintf("Комиссия: %v %v \n", dataForClient.WithDrawFee, dataForClient.Symbol)
		textWithDeal += fmt.Sprintf("Объем допустимый: %v %v \n", dataForClient.WithdrawMax, dataForClient.Symbol)
		textWithDeal += fmt.Sprintf("Сеть: %v \n", dataForClient.Chain)
		textWithDeal += fmt.Sprintf("Объем: %.4f %v \n", dataForClient.AmountCoin, dataForClient.Symbol)
		textWithDeal += fmt.Sprintf("Кол-во ордеров: %v \n", dataForClient.AmountAskOrder)
		textWithDeal += fmt.Sprintf("Стоимость: %.2f USDT \n", dataForClient.AskCost)
		textWithDeal += fmt.Sprintf("Ордера (Цена/Кол-во): %v \n", dataForClient.AskOrder)
		textWithDeal += fmt.Sprintf("📗|%v| \n", dataForClient.MarketTo)
		textWithDeal += fmt.Sprintf("Кол-во ордеров: %v \n", dataForClient.AmountBidOrder)
		textWithDeal += fmt.Sprintf("Стоимость: %.2f USDT \n", dataForClient.BidCost)
		textWithDeal += fmt.Sprintf("Ордера (Цена/Кол-во): %v \n", dataForClient.BidOrder)
		textWithDeal += "--- \n"
		textWithDeal += fmt.Sprintf("💰 Спред: %.2f %%", dataForClient.Spread)
		msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, textWithDeal)
		bot.Send(msg)
	}
}



func (b BotUseCase) getWarn() string {

}