package application

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/service"
	"go.dfds.cloud/ccc-exporter/internal/util"
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
		costService:     service.NewConfluentCostService(confluentClient, false),
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
			currentState: ExportStateNeedCosts,
		})
	}
}

func (e *ExporterApplication) setProcesses(exportProcesses []ExportProcess) {
	e.exportProcesses = exportProcesses
}

// TODO: the 4 following functions could be combined - do we need so many states?
func (e *ExporterApplication) fetchCosts(dayTime util.YearMonthDayDate) bool {
	if !e.costService.HasCostsForDate(dayTime) {
		e.costService.FetchAndCacheCosts(dayTime)
		if !e.costService.HasCostsForDate(dayTime) {
			log.Errorf("unable to fetch costs for %s", dayTime)
			return false
		}
	}
	log.Infof("successfully found confluent costs for %s", dayTime)
	return true
}

func (e *ExporterApplication) getPrometheusUsageData(dayTime util.YearMonthDayDate) bool {
	_, err := e.gathererService.GetMetricsForDay(dayTime)
	if err != nil {
		log.Warnf("unable to get prometheus usage data for %s: %s", dayTime, err)
		return false
	}
	log.Infof("successfully found prometheus usage data for %s", dayTime)
	return true
}

func (e *ExporterApplication) writeToCsv(dayTime util.YearMonthDayDate) bool {
	metricsData, err := e.gathererService.GetMetricsForDay(dayTime)
	if err != nil {
		return false
	}
	err = e.WriteCSV(metricsData)
	if err != nil {
		log.Errorf("unable to write csv for %s: %s", dayTime, err)
		return false
	}
	log.Infof("successfully wrote csv for %s", dayTime)
	return true
}

func (e *ExporterApplication) putCsvInS3(dayTime util.YearMonthDayDate, s3Config config.S3) bool {
	data, err := e.ReadCsvRaw(dayTime)
	if err != nil {
		log.Errorf("unable to find csv locally %s", dayTime, err)
		return false
	}
	err = e.s3Client.PutObject(s3Config.BucketName, fmt.Sprintf("%s/%s", s3Config.BucketKey, dayTime.ToFileNameFormat()), data)
	if err != nil {
		log.Errorf("unable to put csv in s3 for %s: %s", dayTime, err)
		return false
	}
	log.Infof("successfully put csv in s3 for %s", dayTime)
	return true
}

func (e *ExporterApplication) removeDoneProcesses(endedProcessesIndices []int) {
	for id := range endedProcessesIndices {
		e.setProcesses(append(e.exportProcesses[:id], e.exportProcesses[id+1:]...)) //all processes except for #i
	}
}

func (e *ExporterApplication) executeProcessAndUpdateState(process ExportProcess, s3Config config.S3) bool {
	//TODO: are so many states really necessary?
	isDone := false
	switch process.currentState {
	case ExportStateNeedCosts:
		if e.fetchCosts(process.dayTime) {
			process.currentState = ExportStateNeedPrometheusUsageData
		}

	case ExportStateNeedPrometheusUsageData:
		if e.getPrometheusUsageData(process.dayTime) {
			process.currentState = ExportStateNeedLocalCSVExport
		}
	case ExportStateNeedLocalCSVExport:
		if e.writeToCsv(process.dayTime) {
			process.currentState = ExportStateNeedToPutCSVInS3
		}
	case ExportStateNeedToPutCSVInS3:
		if e.putCsvInS3(process.dayTime, s3Config) {
			process.currentState = ExportStateDone
		}
	case ExportStateDone:
		isDone = true
	}
	return isDone
}

func (e *ExporterApplication) processesListFold(s3Config config.S3) {
	ongoingProcesses := []ExportProcess{}
	for _, process := range e.exportProcesses {
		if !e.executeProcessAndUpdateState(process, s3Config) {
			ongoingProcesses = append(ongoingProcesses, process)
		}
		//e.removeDoneProcesses(endedProcesses)
	}
	e.setProcesses(ongoingProcesses)
}

func (e *ExporterApplication) Work(config config.Worker, s3Config config.S3) {
	sleepInterval, err := time.ParseDuration(fmt.Sprintf("%ds", config.IntervalSeconds))
	if err != nil {
		panic(err)
	}
	for {
		if len(e.exportProcesses) == 0 {
			e.SetupProcesses(config.CheckForExportedDataInS3, config.DaysToLookBack)
		}
		e.processesListFold(s3Config)
		time.Sleep(sleepInterval)
		log.Infof("woke up, checking for work")
	}
}
