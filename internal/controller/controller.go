package controller

import (
	"crypto_pro/internal/domain/entity"
)

type MarkerController interface {
}

type Server interface {
	GetSpotHandler(usdt, spread float64) []entity.Transaction
}
