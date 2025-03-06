package config

import (
	"log"

	"github.com/spf13/viper"
)

type MongoConfig struct {
	URI      string
	Database string
	Timeout  int
}

func LoadMongoConfig() MongoConfig {
	viper.AutomaticEnv() // enable overwrite envs

	// default
	viper.SetDefault("mongo.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongo.database", "testdb")
	viper.SetDefault("mongo.timeout", 10)

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("no config file found, use default congifuration: %v", err)
	}

	return MongoConfig{
		URI:      viper.GetString("mongo.uri"),
		Database: viper.GetString("mongo.database"),
		Timeout:  viper.GetInt("mongo.timeout"),
	}
}
