package http

import (
	"crypto_pro/internal/controller"
	"crypto_pro/internal/domain/entity"
	"crypto_pro/internal/domain/usecase"
	"crypto_pro/pkg/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/viper"
)

var _ controller.Server = (*Server)(nil)

type Server struct {
	cfg         viper.Viper
	log         logger.Logger
	client      *http.Client
	taskUseCase usecase.TaskUseCase
}

func New(cfg viper.Viper, log logger.Logger, taskUseCase usecase.TaskUseCase) Server {
	server := Server{
		client:      &http.Client{},
		cfg:         cfg,
		log:         log,
		taskUseCase: taskUseCase,
	}
	return server
}

func (s Server) GetSpotHandler(usdt, spread float64) []entity.Transaction {
	url := fmt.Sprintf("%s?usdt=%f&spread=%f", s.cfg.GetString("endpoint.spot"), usdt, spread)
	response, err := s.client.Get(url)
	if err != nil {
		s.log.Error("failed to get spot", s.log.ErrorC(err))
		return nil
	}
	defer response.Body.Close()

	bodyResponse, err := io.ReadAll(response.Body)
	if err != nil {
		s.log.Error("failed to read body response", s.log.ErrorC(err))
		return nil
	}

	transactions := s.transactionUnmarshal(s.log, bodyResponse)
	return transactions.toEntity()
}

func (s Server) transactionUnmarshal(log logger.Logger, body []byte) transactions {
	response := transactions{}
	err := json.Unmarshal(body, &response)
	if err != nil {
		log.Error("Error when unmarshal data", log.ErrorC(err))
	}
	return response
}
