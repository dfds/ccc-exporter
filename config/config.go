package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Worker struct {
	IntervalSeconds int `mapstructure:"intervalSeconds"`
	DaysToLookBack  int `mapstructure:"daysToLookBack"`
}

type Confluent struct {
	ApiKeyId     string `mapstructure:"apiKeyId" env:"CCC_EXPORTER_CC_API_KEY_ID"`
	ApiKeySecret string `mapstructure:"apiKeySecret" env:"CCC_EXPORTER_CC_API_KEY_SECRET"`
}

type Config struct {
	Worker     Worker    `mapstructure:"worker"`
	Confluent  Confluent `mapstructure:"confluent"`
	Prometheus struct {
		Endpoint string `mapstructure:"endpoint"`
	}
}

func LoadConfig(configName string) (Config, error) {
	var conf Config

	viper.SetConfigName(configName)
	viper.SetConfigType("json")
	viper.AddConfigPath(".")

	viper.SetDefault("worker.intervalSeconds", 60)
	viper.SetDefault("worker.daysToLookBack", 7)

	// override with environment variables if any available
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return conf, fmt.Errorf("error reading configuration file: %w", err)
	}
	if err := viper.Unmarshal(&conf); err != nil {
		return conf, fmt.Errorf("error unmarshaling configuration: %w", err)
	}

	return conf, nil
}
