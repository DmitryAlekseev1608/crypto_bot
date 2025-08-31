package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

func New(configsPath string) *viper.Viper {
	configs := viper.New()
	configs.SetConfigFile(configsPath)

	if err := configs.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("error read file configuration %v: %v", configsPath, err))
	}

	configs.AutomaticEnv()

	return configs
}
