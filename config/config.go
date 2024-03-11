package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type Worker struct {
	IntervalSeconds          int  `mapstructure:"intervalSeconds"`
	DaysToLookBack           int  `mapstructure:"daysToLookBack"`
	CheckForExportedDataInS3 bool `mapstructure:"checkForExportedDataInS3"`
}

type Confluent struct {
	ApiKeyId     string `mapstructure:"apiKeyId" env:"CCC_EXPORTER_CC_API_KEY_ID"`
	ApiKeySecret string `mapstructure:"apiKeySecret" env:"CCC_EXPORTER_CC_API_KEY_SECRET"`
}

type S3 struct {
	BucketName string `mapstructure:"bucketName"  env:"CCC_EXPORTER_S3_BUCKET_NAME"`
	BucketKey  string `mapstructure:"bucketKey" env:"CCC_EXPORTER_S3_BUCKET_KEY"`
	Region     string `mapstructure:"region"`
}

type Config struct {
	Worker     Worker    `mapstructure:"worker"`
	S3         S3        `mapstructure:"s3"`
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

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("CCC")
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
