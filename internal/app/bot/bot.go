package chain

import (
	"context"
	"crypto_pro/pkg/logger"

	"github.com/spf13/viper"
)

type App struct {
	ctx             context.Context
	log             logger.Logger
	cfg             viper.Viper
	serviceProvider *serviceProvider
}

func NewApp(ctx context.Context, log logger.Logger, cfg viper.Viper) App {
	return App{
		ctx:             ctx,
		log:             log,
		cfg:             cfg,
		serviceProvider: newServiceProvider(ctx, log, cfg),
	}
}

func (a App) Run() error {
	a.log.Info("Init adapters and repositories")
	a.serviceProvider.setMarketController()
	a.serviceProvider.setDBAdapter()
	defer a.serviceProvider.dbAdapter.Close()

	a.log.Info("init usecase")
	a.serviceProvider.setChainUseCase()

	a.log.Info("all layers was init, run tasks")
	a.serviceProvider.chainUseCase.Run()

	a.log.Info("Have a nice day!")
	return nil
}
