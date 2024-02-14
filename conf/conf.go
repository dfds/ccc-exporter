package conf

import (
	"encoding/json"
	"github.com/kelseyhightower/envconfig"
	"log"
	"os"
)

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

type Data struct {
	Pricing struct {
		Prod struct {
			NetworkTransfer float64 `json:"networkTransfer"`
			Storage         float64 `json:"storage"`
		} `json:"prod"`
		Dev struct {
			NetworkTransfer float64 `json:"networkTransfer"`
			Storage         float64 `json:"storage"`
		} `json:"dev"`
	}
}

func LoadData() Data {
	data, err := os.ReadFile("data.json")
	if err != nil {
		log.Fatal(err)
	}

	var payload Data
	err = json.Unmarshal(data, &payload)
	if err != nil {
		log.Fatal(err)
	}

	return payload
}
