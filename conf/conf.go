package conf

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	WorkerIntervalSeconds int `json:"workerInterval"`
	Prometheus            struct {
		Endpoint string `json:"endpoint"`
	}
	DaysToLookBack int `json:"daysToLookBack"`
}

const APP_CONF_PREFIX = "CCC_EXPORTER"

func LoadConfig() (Config, error) {
	var conf Config
	err := envconfig.Process(APP_CONF_PREFIX, &conf)

	if conf.WorkerIntervalSeconds == 0 {
		conf.WorkerIntervalSeconds = 60
	}

	return conf, err
}
