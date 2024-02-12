package main

import (
	"fmt"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.dfds.cloud/ccc-exporter/conf"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/metrics"
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
		gatherer.GetAllMetrics()

		fmt.Println("New SSO metrics published")
		time.Sleep(sleepInterval)
	}
}
