package telegram

import (
	"context"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/pkg/logger"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramController struct {
	log         logger.Logger
	bot         *tgbotapi.BotAPI
	taskUseCase usecase.TaskUseCase
	infoUseCase usecase.InfoUseCase
}

func New(log logger.Logger, taskUseCase usecase.TaskUseCase, infoUseCase usecase.InfoUseCase) TelegramController {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	log.Info("Authorized on account", log.StringC("UserName", bot.Self.UserName))
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return TelegramController{log: log, bot: bot, taskUseCase: taskUseCase, infoUseCase: infoUseCase}
}

func (t TelegramController) Run(ctx context.Context) {
	var activeSessions = make(map[int64]context.CancelFunc)
	updates := t.bot.GetUpdatesChan(tgbotapi.NewUpdate(0))
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Инструкция"),
		),
	)
	re := regexp.MustCompile(`^\d+\s\d+(\.\d+)?$`)
	semathore := make(chan struct{}, 30)
	defer close(semathore)
	t.taskUseCase.TrancateRawTransactions()
	t.taskUseCase.TrancateDwhTransactions()

	for update := range updates {
		if update.Message != nil {
			t.log.Info("Received message", t.log.StringC("Message", update.Message.Text),
				t.log.Int64C("ChatID", update.Message.Chat.ID), t.log.IntC("Semaphore", len(semathore)))
			if _, exists := activeSessions[update.Message.Chat.ID]; exists {
				t.sendMessage("Сессия активна", update, keyboard)
				continue
			}

			switch {
			case update.Message.Text == "Инструкция":
				t.sendMessage(t.getInstruction(), update, keyboard)
			case re.MatchString(update.Message.Text):
				ctx, cancel := context.WithCancel(context.Background())
				activeSessions[update.Message.Chat.ID] = cancel
				go func() {
					semathore <- struct{}{}
					defer func() { <-semathore }()
					for {
						select {
						case <-ctx.Done():
							return
						default:
							t.handleRequest(update.Message)
						}
					}
				}()
				t.sendMessage("Сессия начата. Отправьте 's' для отмены.", update, keyboard)

			case update.Message.Text == "s":
				if cancel, exists := activeSessions[update.Message.Chat.ID]; exists {
					cancel()
					delete(activeSessions, update.Message.Chat.ID)
					t.taskUseCase.DeleteSession(update.Message.Chat.ID)
					t.sendMessage("Сессия отменена.", update, keyboard)
				} else {
					t.sendMessage("Нет активной сессии.", update, keyboard)
				}

			default:
				t.sendMessage("Такого действия ботом не предусмотрено или что-то было введено не верно", update, keyboard)
			}

		} else if update.CallbackQuery != nil {
			t.sendInfo(update.CallbackQuery)
		}
	}
}

func (t TelegramController) getInstruction() string {
	return `
Привет! Я чат-бот для биржевой аналитики CryptoPro. Моя основная задача помогать находить наиболее выгодные биржевые транзакции для
спотовых продаж. Я постоянно развиваюсь. На текущий момент я умею работать только с биржами BYBIT и KUKOIN. Просто введи сумму необходимого
количества USDT (целое) и spread (до одного знака после запятой) в % через пробел (пример 100 0.3), чтобы я мог искать для тебя транзакции.
Для остановки режима сканирования бирж отправь s в чат и смотри последнее полученное сообщение, нажми на интересующую сделку и получишь
всю необходимую информацию по ней.
`
}

func (t TelegramController) sendMessage(text string, update tgbotapi.Update,
	keyboard tgbotapi.ReplyKeyboardMarkup) {

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	t.bot.Send(msg)
}

func (t TelegramController) handleRequest(update *tgbotapi.Message) {
	transactions := t.taskUseCase.HandleRequest(update.Text)
	buttons := []tgbotapi.InlineKeyboardButton{}

	for _, transaction := range transactions {
		textOnButton := fmt.Sprintf("%v: %.2f (%.2f%%) 🟢", transaction.Symbol,
			transaction.AmountCoin, transaction.Spread)
		keyMsg := strconv.FormatInt(update.Chat.ID, 10) + "/" + transaction.MarketFrom + "/" +
			transaction.MarketTo + "/" + transaction.Symbol
		button := tgbotapi.NewInlineKeyboardButtonData(textOnButton, keyMsg)
		buttons = append(buttons, button)
	}
	inlineKeyboard := t.createInlineKeyboard(buttons)
	msg := tgbotapi.NewMessage(update.Chat.ID, "Выберите подходящую Вам сделку:")
	msg.ReplyMarkup = inlineKeyboard
	t.bot.Send(msg)
}

func (t TelegramController) createInlineKeyboard(buttons []tgbotapi.InlineKeyboardButton
	) tgbotapi.InlineKeyboardMarkup {

	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, button := range buttons {
		row = append(row, button)
		rows = append(rows, row)
		row = nil
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (t TelegramController) sendInfo(update *tgbotapi.CallbackQuery) {
	callbackQuery := update.CallbackQuery
	logger.Infof("User pressed: %v", callbackQuery.Data)
	





	dataFromRedis := t.redis.GetOrder(1, callbackQuery.Data)
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
		t.bot.Send(msg)
}
}