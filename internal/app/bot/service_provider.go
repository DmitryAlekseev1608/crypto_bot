package bot

import (
	"context"
	"crypto_pro/internal/adapters"
	"crypto_pro/internal/adapters/postgres"
	"crypto_pro/internal/controller"
	"crypto_pro/internal/controller/http"
	"crypto_pro/internal/controller/telegram"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/internal/domain/usecase/task"

	"crypto_pro/pkg/logger"

	"github.com/spf13/viper"
)

type serviceProvider struct {
	ctx                context.Context
	log                logger.Logger
	cfg                viper.Viper
	serverController   controller.Server
	dbAdapter          adapters.DbAdapter
	telegramController controller.TelegramController
	taskUseCase        usecase.TaskUseCase
}

func newServiceProvider(ctx context.Context, log logger.Logger, cfg viper.Viper) *serviceProvider {
	return &serviceProvider{
		ctx: ctx,
		log: log,
		cfg: cfg,
	}
}

func (s *serviceProvider) setServerController() controller.Server {
	if s.serverController == nil {
		serverController := http.New(s.cfg, s.log, s.taskUseCase)
		s.serverController = serverController
	}
	return s.serverController
}

func (s *serviceProvider) setDBAdapter() adapters.DbAdapter {
	if s.dbAdapter == nil {
		dbAdapter := postgres.New(s.ctx, s.cfg, s.log)
		s.dbAdapter = dbAdapter
	}
	return s.dbAdapter
}

func (s *serviceProvider) setTaskUseCase() usecase.TaskUseCase {
	if s.taskUseCase == nil {
		taskUseCase := task.New(s.log, s.serverController, s.dbAdapter)
		s.taskUseCase = taskUseCase
	}
	return s.taskUseCase
}

func (s *serviceProvider) setTelegramController() controller.TelegramController {
	if s.telegramController == nil {
		telegramController := telegram.New(s.log, s.taskUseCase)
		s.telegramController = telegramController
	}
	return s.telegramController
}
