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
	host        string
	taskUseCase usecase.TaskUseCase
}

func New(cfg viper.Viper, log logger.Logger, taskUseCase usecase.TaskUseCase) Server {
	host := cfg.GetString("endpoint.spot_local")
	server := Server{
		client:      &http.Client{},
		host:        host,
		cfg:         cfg,
		log:         log,
		taskUseCase: taskUseCase,
	}

	if err := server.checkLocalEndpoint(); err != nil {
		log.Info("failed to check local endpoint", log.ErrorC(err))
		host = cfg.GetString("endpoint.spot_remote")
	}

	server = server.setHost(host)
	return server
}

func (s Server) setHost(host string) Server {
	s.host = host
	return s
}

func (s Server) checkLocalEndpoint() error {
	url := s.cfg.GetString("endpoint.spot_local")
	response, err := s.client.Get(url)
	if err != nil {
		s.log.Error("failed to get spot", s.log.ErrorC(err))
		return err
	}
	defer response.Body.Close()
	return nil
}

func (s Server) GetSpotHandler(usdt, spreadMin, spreadMax float64) []entity.Transaction {
	url := fmt.Sprintf("%s?usdt=%f&spread_min=%f&spread_max=%f", s.host, usdt, spreadMin, spreadMax)
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
