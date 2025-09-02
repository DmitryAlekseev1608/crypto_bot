package main

import (
	"context"
	"crypto_pro/internal/app/bot"
	"crypto_pro/internal/configs"
	"crypto_pro/pkg/logger"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("Try get configs file path from env")

	configsPath, exists := os.LookupEnv("CONFIGS_APP")
	if !exists {
		panic("Сonfigs app file path is out")
	}

	cfg := configs.New(configsPath)

	logger := logger.New(cfg.GetBool("log_to_file"))

	debug := cfg.GetBool("debug")

	if !debug {
		go func() {
			err := http.ListenAndServe(":6060", nil)
			if err != nil {
				logger.Panic("Error starting pprof server: %v", logger.ErrorC(err))
			}
		}()

		go func() {
			http.Handle("/metrics", promhttp.Handler())
			err := http.ListenAndServe(":8080", nil)
			if err != nil {
				logger.Panic("Error starting Prometheus server: %v", logger.ErrorC(err))
			}
		}()
	}

	app := bot.NewApp(context.Background(), logger, *cfg)
	app.Run()
}
