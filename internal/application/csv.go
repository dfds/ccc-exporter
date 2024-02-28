package application

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/util"
	"os"
	"path/filepath"
)
import "encoding/csv"

const CostsExportDir = "export"

func getFilenameForDay(date util.YearMonthDayDate) string {
	return fmt.Sprintf("%d_%d_%d.csv", date.Year, date.Month, date.Day)
}

func (e *ExporterApplication) EnsureCSVDataFolderExists() error {
	err := os.MkdirAll(CostsExportDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func calcCost(m model.MetricData, costs model.KafkaConfluentCost) float64 {
	inGB := m.Value * 1024 * 1024 * 1024
	switch costs.CostUnit {
	case model.GB:
		return inGB * costs.CostPerUnit
	case model.GBHour:
		if costs.CostType == model.CostTypeKafkaStorage {
			return inGB * costs.CostPerUnit * 24 * model.ConfluentCostKafkaStorageReplicationFactor
		}
		return inGB * costs.CostPerUnit * 24
	}
	return 0
}

func (e *ExporterApplication) TryAddLine(writer *csv.Writer, data model.MetricsDataForDay, clusterId model.ClusterId, metricsKey model.MetricKey) {
	metricData, ok := data.Topics[metricsKey][clusterId]
	if !ok {
		log.Warnf("No data found for cluster %s and metric %s", clusterId, metricsKey)
		return
	}
	costType := metricsKey.ToConfluentCostType()
	costs, err := e.costService.GetKafkaCosts(data.DayDate, clusterId, costType)
	if err != nil {
		log.Warnf("No cost found for cluster %s and cost type %s", clusterId, costType)
		return
	}

	for topic, m := range metricData {
		err = writer.Write([]string{
			data.DayDate.ToCSVString(),
			fmt.Sprintf("%s-%s-%s", clusterId, topic, metricsKey.ToCsvFormatString()),
			fmt.Sprintf("%f", calcCost(m, costs)),
		})
	}
}

func (e *ExporterApplication) ReadCsvRaw(date util.YearMonthDayDate) ([]byte, error) {
	byteData, err := os.ReadFile(getFilenameForDay(date))
	if err != nil {
		return nil, err
	}
	return byteData, nil
}

func (e *ExporterApplication) WriteCSV(data model.MetricsDataForDay) error {

	err := e.EnsureCSVDataFolderExists()
	if err != nil {
		return err
	}

	pathToFile := filepath.Join(CostsExportDir, getFilenameForDay(data.DayDate))
	_, err = os.Stat(pathToFile)
	if err == nil {
		return fmt.Errorf("file %s already exists", pathToFile)
	}

	dataFile, err := os.Create(pathToFile)
	if err != nil {
		return err
	}
	defer dataFile.Close()

	writer := csv.NewWriter(dataFile)
	defer writer.Flush()
	headers := []string{"UsageDate", "ServiceName", "Cost"}
	err = writer.Write(headers)
	if err != nil {
		return err
	}
	for _, clusterId := range model.ConfluentClusters {
		e.TryAddLine(writer, data, clusterId, model.ConfluentKafkaServerReceivedBytes)
		e.TryAddLine(writer, data, clusterId, model.ConfluentKafkaServerSentBytes)
		e.TryAddLine(writer, data, clusterId, model.ConfluentKafkaServerRetainedBytes)
	}

	return nil
}

func (e *ExporterApplication) HasExportedDataForDay(date util.YearMonthDayDate) bool {

	err := e.EnsureCSVDataFolderExists()
	if err != nil {
		return false
	}

	pathToFile := filepath.Join(CostsExportDir, getFilenameForDay(date))
	_, err = os.Stat(pathToFile)
	return err == nil
}
