package application

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/service"
	"go.dfds.cloud/ccc-exporter/internal/util"
	"time"
)

type ExportState string

const (
	ExportStateNeedCosts               ExportState = "NEED_COSTS"
	ExportStateNeedPrometheusUsageData ExportState = "NEED_PROMETHEUS_USAGE_DATA"
	ExportStateNeedLocalCSVExport      ExportState = "NEED_LOCAL_CSV_EXPORT"
	ExportStateNeedToPutCSVInS3        ExportState = "NEED_TO_PUT_CSV_IN_S3"
	ExportStateDone                    ExportState = "DONE"
)

type ExportProcess struct {
	dayTime      util.YearMonthDayDate
	currentState ExportState
}

// ExporterApplication responsible for using various clients and services to be able to create a csv with confluent costs and output them in a s3 bucket in AWS
type ExporterApplication struct {
	gathererService *service.GathererService
	costService     *service.ConfluentCostService
	s3Client        *client.S3Client

	exportProcesses []ExportProcess
}

func NewExporterApplication(prometheusClient *client.PrometheusClient, confluentClient *client.ConfluentCloudClient, s3Client *client.S3Client) ExporterApplication {

	return ExporterApplication{
		gathererService: service.NewGatherer(prometheusClient),
		costService:     service.NewConfluentCostService(confluentClient, true),
		s3Client:        s3Client,
	}
}

// SetupProcesses setup fetch processes for days looking back by daysToLookBack
// For now only checks local exports and not in s3
func (e *ExporterApplication) SetupProcesses(checkS3 bool, daysToLookBack int) {
	e.exportProcesses = []ExportProcess{}

	var daysToExport []util.YearMonthDayDate
	year, month, day := time.Now().UTC().Date()
	date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	for i := 0; i < daysToLookBack; i++ {
		date = date.Add(-time.Hour * 24)
		daysToExport = append(daysToExport, util.ToYearMonthDayDate(date))
	}

	if checkS3 {
		log.Errorf("checking s3 for exported data is not implemented yet")
	}

	log.Infof("checking locally for exported data for the last %d days", daysToLookBack)
	for _, yearMonthDayDate := range daysToExport {
		if e.HasExportedDataForDay(yearMonthDayDate) {
			continue
		}
		e.exportProcesses = append(e.exportProcesses, ExportProcess{
			dayTime:      yearMonthDayDate,
			currentState: ExportStateNeedPrometheusUsageData,
		})
	}
}

func (e *ExporterApplication) Work(config config.Worker, s3Config config.S3) {
	sleepInterval, err := time.ParseDuration(fmt.Sprintf("%ds", config.IntervalSeconds))
	if err != nil {
		panic(err)
	}

	for {
		if len(e.exportProcesses) == 0 {
			e.SetupProcesses(config.CheckForExportedDataInS3, config.DaysToLookBack)
			if len(e.exportProcesses) == 0 {
				log.Infof("no more work to do, sleeping for %s", sleepInterval)
				time.Sleep(sleepInterval)
				continue
			}
		}

		// TODO: Very messy state machine, should be refactored
		for i, process := range e.exportProcesses {
			switch process.currentState {
			case ExportStateNeedCosts:
				if !e.costService.HasCostsForDate(process.dayTime) {
					e.costService.FetchAndCacheCosts(process.dayTime)
					if !e.costService.HasCostsForDate(process.dayTime) {
						log.Errorf("unable to fetch costs for %s", process.dayTime)
						continue
					}
				}
				log.Infof("successfully found confluent costs for %s", process.dayTime)
				e.exportProcesses[i].currentState = ExportStateNeedPrometheusUsageData
			case ExportStateNeedPrometheusUsageData:
				_, err := e.gathererService.GetMetricsForDay(process.dayTime)
				if err != nil {
					log.Warnf("unable to get prometheus usage data for %s: %s", process.dayTime, err)
					continue
				}
				log.Infof("successfully found prometheus usage data for %s", process.dayTime)
				e.exportProcesses[i].currentState = ExportStateNeedLocalCSVExport
			case ExportStateNeedLocalCSVExport:
				metricsData, err := e.gathererService.GetMetricsForDay(process.dayTime)
				if err != nil {
					return
				}
				err = e.WriteCSV(metricsData)
				if err != nil {
					log.Errorf("unable to write csv for %s: %s", process.dayTime, err)
					continue
				}
				log.Infof("successfully wrote csv for %s", process.dayTime)
				e.exportProcesses[i].currentState = ExportStateNeedToPutCSVInS3
			case ExportStateNeedToPutCSVInS3:
				data, err := e.ReadCsvRaw(process.dayTime)
				if err != nil {
					log.Errorf("unable to find csv locally %s", process.dayTime, err)
					continue
				}
				err = e.s3Client.PutObject(s3Config.BucketName, s3Config.BucketKey, data)
				if err != nil {
					log.Errorf("unable to put csv in s3 for %s: %s", process.dayTime, err)
					continue
				}
				log.Infof("successfully put csv in s3 for %s", process.dayTime)
				e.exportProcesses[i].currentState = ExportStateDone
			case ExportStateDone:
			}

			// remove done processes
			for i := len(e.exportProcesses) - 1; i >= 0; i-- {
				if e.exportProcesses[i].currentState == ExportStateDone {
					e.exportProcesses = append(e.exportProcesses[:i], e.exportProcesses[i+1:]...)
				}
			}
		}
		time.Sleep(sleepInterval)
		log.Infof("woke up, checking for work")
	}
}
