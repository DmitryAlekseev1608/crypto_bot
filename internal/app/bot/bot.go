package bot

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
	a.serviceProvider.setServerController()
	a.serviceProvider.setTelegramController()
	a.serviceProvider.setDBAdapter()
	defer a.serviceProvider.dbAdapter.Close()

	a.log.Info("init usecase")
	a.serviceProvider.setTaskUseCase()

	a.log.Info("all layers was init, run tasks")
	a.serviceProvider.telegramController.Run(a.ctx)

	a.log.Info("Have a nice day!")
	return nil
}
