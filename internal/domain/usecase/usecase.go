package usecase

import (
	"crypto_pro/internal/domain/entity"
)

type TaskUseCase interface {
	HandleRequest(requestIn string, id int64) []entity.Transaction
	DeleteSession(id int64)
	TrancateRawTransactions()
	TrancateDwhTransactions()
	GetInfoAboutTransactions(id int64, marketFrom, marketTo, symbol string) string
	GetTransactions(id int64) []entity.Transaction
	GetInstruction() string
}
