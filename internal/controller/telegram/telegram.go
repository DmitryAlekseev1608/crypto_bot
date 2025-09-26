package telegram

import (
	"context"
	"crypto_pro/internal/controller"
	"crypto_pro/internal/domain/entity"
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

var _ controller.TelegramController = (*TelegramController)(nil)

type TelegramController struct {
	log         logger.Logger
	bot         *tgbotapi.BotAPI
	updates     tgbotapi.UpdateConfig
	taskUseCase usecase.TaskUseCase
}

func New(log logger.Logger, taskUseCase usecase.TaskUseCase) TelegramController {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_APITOKEN"))
	if err != nil {
		panic(err)
	}
	bot.Debug = false
	log.Info("Authorized on account", log.StringC("UserName", bot.Self.UserName))

	_, err = bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		log.Error("Failed to delete webhook (this is normal if no webhook was set)", log.ErrorC(err))
	} else {
		log.Info("Successfully deleted existing webhook")
	}

	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 60

	return TelegramController{log: log, bot: bot, taskUseCase: taskUseCase, updates: updates}
}

func (t TelegramController) Run(ctx context.Context) {
	var activeSessions = make(map[int64]clientUpdate)

	updates := t.bot.GetUpdatesChan(t.updates)
	keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Инструкция")))
	re := regexp.MustCompile(`^\d+\s\d+(\.\d+)?$`)
	semathore := make(chan struct{}, 30)
	defer close(semathore)
	t.taskUseCase.TrancateRawTransactions()
	t.taskUseCase.TrancateDwhTransactions()

	for update := range updates {
		if update.Message != nil {
			t.log.Info("Received message", t.log.StringC("Message", update.Message.Text),
				t.log.Int64C("ChatID", update.Message.Chat.ID), t.log.IntC("Semaphore",
					len(semathore)))

			switch {
			case update.Message.Text == "Инструкция":
				t.sendMessage(t.taskUseCase.GetInstruction(), update, keyboard)
			case re.MatchString(update.Message.Text):
				if _, exists := activeSessions[update.Message.Chat.ID]; exists {
					t.sendMessage("Сессия активна", update, keyboard)
					continue
				}

				t.taskUseCase.CreateSession(strconv.Itoa(int(update.Message.Chat.ID)), update.Message.Text)
				ctx, cancelFunc := context.WithCancel(context.Background())
				activeSessions[update.Message.Chat.ID] = clientUpdate{
					cancelFunc: cancelFunc,
					time:       time.Now(),
				}

				semathore <- struct{}{}
				go func() {
					defer func() { <-semathore }()
					ticker := time.NewTicker(time.Second * 120)
					for timeT := time.Now(); ; timeT = <-ticker.C {
						_ = timeT

						select {
						case <-ctx.Done():
							return
						default:
							t.handleRequest(update)
						}
					}
				}()
				t.sendMessage("Сессия начата. Отправьте 'stop' для отмены.", update, keyboard)

			case update.Message.Text == "stop":
				if clientUpdate, exists := activeSessions[update.Message.Chat.ID]; exists {
					clientUpdate.cancelFunc()
					delete(activeSessions, update.Message.Chat.ID)
					t.taskUseCase.DeleteSession(strconv.Itoa(int(update.Message.Chat.ID)))
					t.sendMessage("Сессия отменена.", update, keyboard)
				} else {
					t.sendMessage("Нет активной сессии.", update, keyboard)
				}

			case update.Message.Text == "all":
				transactions := t.taskUseCase.GetAllTransactions(strconv.Itoa(int(update.Message.Chat.ID)))
				if len(transactions) == 0 {
					t.sendMessage("Нет транзакций.", update, keyboard)
					continue
				}
				t.sendAllButtons(transactions, update)

			default:
				t.sendMessage(`Такого действия ботом не предусмотрено или что-то было введено не
					верно`, update, keyboard)
			}

		} else if update.CallbackQuery != nil {
			t.sendInfo(update)
		}
	}
}

func (t TelegramController) sendMessage(text string, update tgbotapi.Update,
	keyboard tgbotapi.ReplyKeyboardMarkup) {

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ReplyMarkup = keyboard
	t.bot.Send(msg)
}

func (t TelegramController) handleRequest(update tgbotapi.Update) {
	transactions := t.taskUseCase.HandleRequest(update.Message.Text, strconv.Itoa(int(update.Message.Chat.ID)))
	if transactions == nil {
		return
	}
	t.sendAllButtons(transactions, update)
}

func (t TelegramController) sendAllButtons(transactions []entity.Transaction, update tgbotapi.Update) {
	buttons := []tgbotapi.InlineKeyboardButton{}

	for _, transaction := range transactions {
		textOnButton := fmt.Sprintf("🟢 %v: %.2f (%.2f%%) 📕 %s -> 📗 %s", transaction.Symbol,
			transaction.AmountCoin, transaction.Spread, transaction.MarketFrom, transaction.MarketTo)
		keyMsg := transaction.MarketFrom + "/" + transaction.MarketTo + "/" + transaction.Symbol
		button := tgbotapi.NewInlineKeyboardButtonData(textOnButton, keyMsg)
		buttons = append(buttons, button)
	}
	inlineKeyboard := t.createInlineKeyboard(buttons)
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите подходящую Вам сделку:")
	msg.ReplyMarkup = inlineKeyboard
	t.bot.Send(msg)
}

func (t TelegramController) createInlineKeyboard(buttons []tgbotapi.InlineKeyboardButton,
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

func (t TelegramController) sendInfo(update tgbotapi.Update) {
	callbackQuery := update.CallbackQuery
	t.log.Info("User pressed button", t.log.StringC("Data", callbackQuery.Data))
	marketFrom, marketTo, symbol := t.getKeyFromUpdate(callbackQuery)
	msgContent := t.taskUseCase.GetInfoAboutTransactions(strconv.Itoa(int(callbackQuery.Message.Chat.ID)), marketFrom,
		marketTo, symbol)
	msg := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, msgContent)
	msg.ParseMode = "Markdown"
	t.bot.Send(msg)
}

func (t TelegramController) getKeyFromUpdate(update *tgbotapi.CallbackQuery) (string, string,
	string) {
	parts := strings.Split(update.Data, "/")
	return parts[0], parts[1], parts[2]
}

func (t TelegramController) DeleteWebhook() error {
	_, err := t.bot.Request(tgbotapi.DeleteWebhookConfig{})
	if err != nil {
		return err
	}
	t.log.Info("Webhook successfully deleted")
	return nil
}
