package info

import (
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/pkg/logger"
)

var _ usecase.InfoUseCase = (*InfoUseCase)(nil)

type InfoUseCase struct {
	log       logger.Logger
	dbAdapter adapters.DbAdapter
}

func New(log logger.Logger, dbAdapter adapters.DbAdapter) InfoUseCase {
	return InfoUseCase{log: log, dbAdapter: dbAdapter}
}

func (i InfoUseCase) GetTransactions(id int64) []entity.Transaction {
	return i.dbAdapter.SelectTransactions(id)
}
