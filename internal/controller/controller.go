package controller

import (
	"crypto_pro/internal/configs"
	"crypto_pro/internal/domain/entity"

	"github.com/labstack/echo/v4"
)

type MarkerController interface {
	GetChain(market configs.Market) []entity.Chain
	GetOrder(market configs.Market, symbol string) entity.Orders
}

type Server interface {
	GetSpot(c echo.Context) error
}
