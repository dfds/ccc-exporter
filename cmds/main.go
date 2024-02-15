package main

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.dfds.cloud/ccc-exporter/conf"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/metrics"
	"log"
	"time"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(pprof.New())

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	go worker()
	err := app.Listen(":8080")
	if err != nil {
		panic(err)
	}
}

func worker() {
	config, err := conf.LoadConfig()
	if err != nil {
		panic(err)
	}

	sleepInterval, err := time.ParseDuration(fmt.Sprintf("%ds", config.WorkerInterval))
	if err != nil {
		panic(err)
	}

	for {
		fmt.Println("Getting Confluent Cloud cost data")

		promClient := client.NewClient(config.Prometheus.Endpoint)
		gatherer := metrics.NewGatherer(promClient)
		data := gatherer.GetAllMetrics()

		metricsByCaps := metrics.ByCapability(data)

		dataSum := 0.0
		keyCount := 0

		for _, next := range metricsByCaps {
			keyCount = keyCount + 1
			for _, next2 := range next {
				keyCount = keyCount + 1
				for metricKey, val := range next2 {
					keyCount = keyCount + 1
					if metricKey == metrics.ConfluentKafkaServerRetainedBytes {
						dataSum = dataSum + (val / 1024 / 1024 / 1024)
					}
				}
			}
		}

		fmt.Printf("Main::metricsByCaps end sum: %f\n", dataSum)
		fmt.Printf("Main:: keyCount: %d\n", keyCount)

		pricingData := conf.LoadData()

		pricingProd := metrics.Pricing{
			NetworkTransfer: pricingData.Pricing.Prod.NetworkTransfer,
			Storage:         pricingData.Pricing.Prod.Storage,
		}

		pricingDev := metrics.Pricing{
			NetworkTransfer: pricingData.Pricing.Dev.NetworkTransfer,
			Storage:         pricingData.Pricing.Dev.Storage,
		}

		csvData := metrics.CapabilityResponseToCostCsv(metricsByCaps, pricingProd, pricingDev)

		serialised, err := json.Marshal(csvData)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(string(serialised))

		fmt.Println("New metrics published")
		time.Sleep(sleepInterval)
	}
}
