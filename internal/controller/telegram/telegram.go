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
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TelegramController struct {
	log        logger.Logger
	bot        *tgbotapi.BotAPI
	botUseCase usecase.BotUseCase
}

func New(log logger.Logger) TelegramController {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	log.Info("Authorized on account", log.StringC("UserName", bot.Self.UserName))
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	return TelegramController{log: log, bot: bot}
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
							t.handleRequest(ctx, update.Message)
						}
					}
				}()
				t.sendMessage("Сессия начата. Отправьте 's' для отмены.", update, keyboard)

			case update.Message.Text == "s":
				if cancel, exists := activeSessions[update.Message.Chat.ID]; exists {
					cancel()
					delete(activeSessions, update.Message.Chat.ID)
					t.sendMessage("Сессия отменена.", update, keyboard)
				} else {
					t.sendMessage("Нет активной сессии.", update, keyboard)
				}

			default:
				t.sendMessage("Такого действия ботом не предусмотрено или что-то было введено не верно", update, keyboard)
			}

		} else if update.CallbackQuery != nil {
			callbackQuery := update.CallbackQuery
			t.log.Info("Received callback query", t.log.StringC("CallbackQuery", callbackQuery.Data),
				t.log.Int64C("ChatID", callbackQuery.Message.Chat.ID))
			t.botUseCase.GetTransactions(ctx, callbackQuery.Data)
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

func (t TelegramController) sendMessage(text string, update tgbotapi.Update, keyboard tgbotapi.ReplyKeyboardMarkup) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	t.bot.Send(msg)
}

func (t TelegramController) handleRequest(ctx context.Context, update *tgbotapi.Message) {
	inputFromClient := transformTextInSlice(update.Text)
	usdt := float32(inputFromClient[0])
	spread := float32(inputFromClient[1])
	request := request.RequestParameters{
		URL:   "http://localhost:8080/spot",
		Query: map[string]interface{}{"usdt": usdt, "spread": spread},
	}
	request.Request()
	response := unmarshalDataFromService(request.BodyResponse)
	if len(response) != 0 {
		buttons := []tgbotapi.InlineKeyboardButton{}
		for _, row := range response {
			textOnButton := fmt.Sprintf("%v: %.2f (%.2f%%) 🟢", row.Symbol, row.AmountCoin, row.Spread)
			keyMsg := strconv.FormatInt(update.Chat.ID, 10) + "/" + row.MarketFrom + "/" + row.MarketTo + "/" + row.Symbol
			clientInfoVal := redis.ClientInfoVal{}
			clientInfoVal.Symbol = row.Symbol
			clientInfoVal.MarketFrom = row.MarketFrom
			clientInfoVal.WithDrawFee = row.WithDrawFee
			clientInfoVal.WithdrawMax = row.WithdrawMax
			clientInfoVal.Chain = row.Chain
			clientInfoVal.AmountCoin = row.AmountCoin
			clientInfoVal.AmountAskOrder = row.AmountAskOrder
			clientInfoVal.AskCost = row.AskCost
			clientInfoVal.AskOrder = transformInFormForREST(row.AskOrder)
			clientInfoVal.MarketTo = row.MarketTo
			clientInfoVal.AmountBidOrder = row.AmountBidOrder
			clientInfoVal.BidCost = row.BidCost
			clientInfoVal.BidOrder = transformInFormForREST(row.BidOrder)
			clientInfoVal.Spread = row.Spread
			var textUnderButton = make(redis.ClientInfo)
			textUnderButton[keyMsg] = clientInfoVal
			button := tgbotapi.NewInlineKeyboardButtonData(textOnButton, keyMsg)
			buttons = append(buttons, button)
			textUnderButton.Insert()
		}
		inlineKeyboard := createInlineKeyboard(buttons)
		msg := tgbotapi.NewMessage(update.Chat.ID, "Выберите подходящую Вам сделку:")
		msg.ReplyMarkup = inlineKeyboard
		bot.Send(msg)
		time.Sleep(10 * time.Second)
	}
}

func transformTextInSlice(input string) (numbers []float32) {
	parts := strings.Split(input, " ")
	numbers = []float32{}
	for _, part := range parts {
		num, err := strconv.ParseFloat(part, 32)
		if err != nil {
			logger.Warnf("Error when transforming '%s': %v", part, err)
			continue
		}
		numbers = append(numbers, float32(num))
	}
	return
}
