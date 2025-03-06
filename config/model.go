package config

import (
	"github.com/spf13/viper"
	"github.com/world-in-progress/yggdrasil/core/logger"
)

type ModelConfig struct {
	Path string
}

func LoadModelConfig() *ModelConfig {
	viper.AutomaticEnv() // enable overwrite envs

	if err := viper.ReadInConfig(); err != nil {
		logger.Error("no config file found: %v", err)
		return nil
	}

	return &ModelConfig{
		Path: viper.GetString("model.path"),
	}
}
