package application

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/service"
	"time"
)

type ExportState string

const (
	ExportStateNeedCosts               ExportState = "NEED_COSTS"
	ExportStateNeedPrometheusUsageData ExportState = "NEED_PROMETHEUS_USAGE_DATA"
	ExportStateDone                    ExportState = "DONE"
)

type ExportProcess struct {
	dayTime      time.Time
	currentState ExportState
}

// ExporterApplication responsible for using various clients and services to be able to create a csv with confluent costs and output them in a s3 bucket in AWS
type ExporterApplication struct {
	gathererService *service.GathererService
	costService     *service.ConfluentCostService

	exportProcesses []ExportProcess
	confluentClient *client.ConfluentCloudClient
}

func NewExporterApplication(prometheusClient *client.PrometheusClient, confluentClient *client.ConfluentCloudClient) ExporterApplication {

	return ExporterApplication{
		gathererService: service.NewGatherer(prometheusClient),
		costService:     service.NewConfluentCostService(confluentClient, true),
	}
}

// SetupProcesses setup fetch processes for days looking back by checking if we have data locally for those days
func (e *ExporterApplication) SetupProcesses(daysToLookBack int) {

	e.exportProcesses = []ExportProcess{}

	year, month, day := time.Now().UTC().Date()
	now := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= daysToLookBack; i++ {
		date := now.Add(-time.Hour * time.Duration(i))
		if e.HasExportedDataForDay(date) {
			continue
		}
		e.exportProcesses = append(e.exportProcesses, ExportProcess{
			dayTime:      date,
			currentState: ExportStateNeedCosts,
		})
	}
}

func (e *ExporterApplication) Work(config config.Worker) {
	sleepInterval, err := time.ParseDuration(fmt.Sprintf("%ds", config.IntervalSeconds))
	if err != nil {
		panic(err)
	}

	for {
		e.SetupProcesses(config.DaysToLookBack)
		if len(e.exportProcesses) == 0 {
			fmt.Println("No data to be exported")
			time.Sleep(sleepInterval)
			continue
		}

		for i, process := range e.exportProcesses {
			switch process.currentState {
			case ExportStateNeedCosts:
				if !e.costService.HasCostsForDate(process.dayTime) {
					continue
				}
				e.exportProcesses[i].currentState = ExportStateNeedPrometheusUsageData
			case ExportStateNeedPrometheusUsageData:
				data, err := e.gathererService.GetMetricsForDay(process.dayTime)
				if err != nil {
					log.Warnf("unable to get prometheus usage data for %s: %s", process.dayTime, err)
					continue
				}
				err = e.WriteCSV(data)
				if err != nil {
					log.Warnf("unable to get write csv usage data for %s: %s", process.dayTime, err)
					continue
				}
				e.exportProcesses[i].currentState = ExportStateDone
				log.Infof("successfully export cost data for %s", process.dayTime)
			}
		}

		time.Sleep(sleepInterval)
	}
}
