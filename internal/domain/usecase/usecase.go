package usecase

import (
	"crypto_pro/internal/domain/entity"
)

type TaskUseCase interface {
	HandleRequest(requestIn string) []entity.Transaction
	DeleteSession(id int64)
	TrancateRawTransactions()
	TrancateDwhTransactions()
}

type InfoUseCase interface {
	GetTransactions(id int64) []entity.Transaction
}
