package usecase

import (
	"context"
	"crypto_pro/internal/domain/usecase/spot"
)

type ChainUseCase interface {
	Run()
}

type OrderUseCase interface {
	Run()
}

type TransactionUseCase interface {
	Run()
}

type SpotUseCase interface {
	Run(price, spread float64) ([]spot.Arbitrage, error)
}

type BotUseCase interface {
	GetTransactions(ctx context.Context, data string)
}
