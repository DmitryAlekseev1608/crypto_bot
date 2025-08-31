package adapters

import (
	"crypto_pro/internal/domain/entity"
)

type DbAdapter interface {
	Close()
	UpsertDWHTransactions(transactions []entity.Transaction) error
	SelectTransactions(id int64) []entity.Transaction
	DeleteSession(id int64)
	TrancateRawTransactions()
	TrancateDwhTransactions()
}
