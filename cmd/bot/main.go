package main

import (
	"context"
	"crypto_pro/pkg/config"
	openapi "crypto_pro/pkg/openapi/go"
	"crypto_pro/pkg/redis"
	"crypto_pro/pkg/request"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()
var activeSessions = make(map[int64]context.CancelFunc)

func main() {
	go func() {
		err := http.ListenAndServe(":6066", nil)
		if err != nil {
			logger.Fatalf("Error starting pprof server: %v", err)
		}
	}()
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(":8086", nil)
		if err != nil {
			logger.Fatalf("Error starting Prometheus server: %v", err)
		}
	}()
	bot, err := tgbotapi.NewBotAPI(config.LoadEnv("TELEGRAM_APITOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = true
	logger.Infof("Authorized on account %v", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Инструкция"),
		),
	)
	re := regexp.MustCompile(`^\d+\s\d+(\.\d+)?$`)
	for update := range updates {
		if update.Message != nil {
			logger.Infof("Received message: %s\n", update.Message.Text)
			switch {
			case update.Message.Text == "Инструкция":
				textInstruction := `
Привет! Я чат-бот для биржевой аналитики CryptoPro. Моя основная задача помогать находить наиболее выгодные биржевые транзакции для
спотовых продаж. Я постоянно развиваюсь. На текущий момент я умею работать только с биржами BYBIT и KUKOIN. Просто введи сумму необходимого
количества USDT (целое) и spread (до одного знака после запятой) в % через пробел (пример 100 0.3), чтобы я мог искать для тебя транзакции.
Для остановки режима сканирования бирж отправь s в чат и смотри последнее полученное сообщение, нажми на интересующую сделку и получишь
всю необходимую информацию по ней.
`
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, textInstruction)
				msg.ReplyMarkup = keyboard
				bot.Send(msg)
			case re.MatchString(update.Message.Text):
				if _, exists := activeSessions[update.Message.Chat.ID]; exists {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сессия активна"))
					continue
				}
				ctx, cancel := context.WithCancel(context.Background())
				activeSessions[update.Message.Chat.ID] = cancel
				go processRequest(ctx, bot, update.Message)
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Сессия начата. Отправьте 's' для отмены."))
			case update.Message.Text == "s":
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
	}
}

func unmarshalDataFromService(body []byte) (response []openapi.SpotGet200ResponseInner) {
	response = []openapi.SpotGet200ResponseInner{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		logger.Warnf("Error when unmarshal data: %v", err)
	}
	return
}

func createInlineKeyboard(buttons []tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	var row []tgbotapi.InlineKeyboardButton
	for _, button := range buttons {
		row = append(row, button)
		rows = append(rows, row)
		row = nil
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
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

func processRequest(ctx context.Context, bot *tgbotapi.BotAPI, update *tgbotapi.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
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
	}
}

func transformInFormForREST(askBid []openapi.SpotGet200ResponseInnerAskOrderInner) (askBidStruct []redis.AskBid) {
	askBidStruct = []redis.AskBid{}
	for _, val := range askBid {
		order := redis.AskBid{}
		order.Price = val.Price
		order.QTy = val.QTy
		askBidStruct = append(askBidStruct, order)
	}
	return
}
