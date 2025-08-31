package adapters

import (
	"crypto_pro/internal/configs"
	"crypto_pro/internal/domain/entity"
)

type DbAdapter interface {
	Close()
	TrancateRowChains() error
	InsertRowChains(data []entity.Chain) error
	UpsertDWHChains() error
	UpsertDWHOrders() error
	SelectSymbols(market configs.Market) []string
	UpdateOrders(data entity.Orders) error
	SelectDWHChains() ([]entity.Chain, error)
	SelectDWHOrders() ([]entity.Orders, error)
}
