package controller

import (
	"context"
	"crypto_pro/internal/domain/entity"
)

type MarkerController interface {
}

type Server interface {
	GetSpotHandler(usdt, spread float64) []entity.Transaction
}

type TelegramController interface {
	Run(ctx context.Context)
}
