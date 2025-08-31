package chain

import (
	"context"
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/adapters/postgres"
	"crypto_pro/internal/controller"
	"crypto_pro/internal/controller/market"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/internal/domain/usecase/chain"

	"crypto_pro/pkg/logger"

	"github.com/spf13/viper"
)

type serviceProvider struct {
	ctx              context.Context
	log              logger.Logger
	cfg              viper.Viper
	marketController controller.MarkerController
	dbAdapter        adapters.DbAdapter
	chainUseCase     usecase.ChainUseCase
}

func newServiceProvider(ctx context.Context, log logger.Logger, cfg viper.Viper) *serviceProvider {
	return &serviceProvider{
		ctx: ctx,
		log: log,
		cfg: cfg,
	}
}

func (s *serviceProvider) setMarketController() controller.MarkerController {
	if s.marketController == nil {
		marketController := market.New(s.cfg, s.log)
		s.marketController = marketController
	}
	return s.marketController
}

func (s *serviceProvider) setDBAdapter() adapters.DbAdapter {
	if s.dbAdapter == nil {
		dbAdapter := postgres.New(s.ctx, s.cfg, s.log)
		s.dbAdapter = dbAdapter
	}
	return s.dbAdapter
}

func (s *serviceProvider) setChainUseCase() usecase.ChainUseCase {
	if s.chainUseCase == nil {
		chainUseCase := chain.New(s.cfg, s.log, s.marketController, s.dbAdapter)
		s.chainUseCase = chainUseCase
	}
	return s.chainUseCase
}
