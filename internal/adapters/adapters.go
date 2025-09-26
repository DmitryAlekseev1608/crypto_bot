package adapters

import (
	"crypto_pro/internal/domain/entity"
)

type DbAdapter interface {
	Close()
	UpsertDWHTransactions(transactions []entity.Transaction) error
	SelectTransactions(id string) []entity.Transaction
	DeleteSession(id string)
	TrancateRawTransactions()
	TrancateDwhTransactions()
	SelectTransactionsBySymbol(id string, symbol, marketFrom, marketTo string) entity.Transaction
	SelectNewTransactions(id string) []entity.Transaction
}
