package usecase

import (
	"crypto_pro/internal/domain/entity"
)

type TaskUseCase interface {
	HandleRequest(requestIn string, id string) []entity.Transaction
	DeleteSession(id string)
	TrancateRawTransactions()
	TrancateDwhTransactions()
	GetInfoAboutTransactions(id string, marketFrom, marketTo, symbol string) string
	GetTransactions(id string) []entity.Transaction
	GetInstruction() string
	GetAllTransactions(id string) []entity.Transaction
	CreateSession(id, requestIn string)
}
