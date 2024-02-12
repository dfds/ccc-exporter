package conf

import "github.com/kelseyhightower/envconfig"

type Config struct {
	WorkerInterval int `json:"workerInterval"`
	Prometheus     struct {
		Endpoint string `json:"endpoint"`
	}
}

const APP_CONF_PREFIX = "CCCE"

func LoadConfig() (Config, error) {
	var conf Config
	err := envconfig.Process(APP_CONF_PREFIX, &conf)

	if conf.WorkerInterval == 0 {
		conf.WorkerInterval = 60
	}

	return conf, err
}
