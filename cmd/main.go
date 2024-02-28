package main

import (
	"context"
	"flag"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/application"
	"go.dfds.cloud/ccc-exporter/internal/client"
)

const defaultConfigFile = "config.json"

var configFile = flag.String("config", "config.json", "Path to configuration file")

func main() {
	flag.Parse()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(pprof.New())
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	if *configFile != defaultConfigFile {
		log.Info().Msgf("Using config file: %s", *configFile)
	}

	loadedConfig, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to load config file: %s", *configFile)
	}
	promClient := client.NewPrometheusClient(loadedConfig.Prometheus.Endpoint)
	confluentClient := client.NewConfluentCloudClient(loadedConfig.Confluent)

	loadedAwsConfig, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load AWS config")
	}

	loadedAwsConfig.Region = loadedConfig.S3.Region
	s3Client, err := client.NewS3Client(loadedAwsConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create S3 client")
	}

	exporterApplication := application.NewExporterApplication(promClient, confluentClient, s3Client)
	go exporterApplication.Work(loadedConfig.Worker, loadedConfig.S3)

	err = app.Listen(":8080")
	if err != nil {
		panic(err)
	}
}
