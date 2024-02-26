package main

import (
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.dfds.cloud/ccc-exporter/conf"
	"go.dfds.cloud/ccc-exporter/internal/application"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/service"
)

func main() {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(pprof.New())

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	config, err := conf.LoadConfig()
	promClient := client.NewPrometheusClient(config.Prometheus.Endpoint)

	exporterApplication := application.NewExporterApplication(promClient, service.NewConfluentCostService(true))
	go exporterApplication.Work(config.WorkerIntervalSeconds, config.DaysToLookBack)

	err = app.Listen(":8080")
	if err != nil {
		panic(err)
	}
}
