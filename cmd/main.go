package main

import (
	"flag"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/application"
	"go.dfds.cloud/ccc-exporter/internal/client"
)

var configFile = flag.String("config", "config.json", "Path to configuration file")

func main() {
	flag.Parse()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(pprof.New())
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	loadedConfig, err := config.LoadConfig(*configFile)
	if err != nil {
		panic(err)
	}
	promClient := client.NewPrometheusClient(loadedConfig.Prometheus.Endpoint)
	confluentClient := client.NewConfluentCloudClient(loadedConfig.Confluent)

	exporterApplication := application.NewExporterApplication(promClient, confluentClient)
	go exporterApplication.Work(loadedConfig.Worker)

	err = app.Listen(":8080")
	if err != nil {
		panic(err)
	}
}
