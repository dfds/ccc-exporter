package application

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/service"
	"os"
	"path/filepath"
	"time"
)
import "encoding/csv"

const CostsExportDir = "exports"

func getFilenameForDay(date time.Time) string {
	year, month, day := date.Date()
	return fmt.Sprintf("%d_%d_%d.csv", year, month, day)
}

func (e *ExporterApplication) EnsureCSVDataFolderExists() error {
	err := os.MkdirAll(CostsExportDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func ToConfluentCostType(m model.MetricKey) service.CostType {
	switch m {
	case model.ConfluentKafkaServerReceivedBytes:
		return service.CostTypeKafkaNetworkRead
	case model.ConfluentKafkaServerSentBytes:
		return service.CostTypeKafkaNetworkWrite
	case model.ConfluentKafkaServerRetainedBytes:
		return service.CostTypeKafkaStorage
	}
	return "INVALID"
}

func calcCost(m service.MetricData, costs service.ConfluentCost) float64 {
	inGB := m.Value * 1024 * 1024 * 1024
	switch costs.CostUnit {
	case service.USDPerGB:
		return inGB * costs.CostPerUnit
	case service.USDPerGBHour:
		if costs.CostType == service.CostTypeKafkaStorage {
			return inGB * costs.CostPerUnit * 24 * service.ConfluentCostKafkaStorageReplicationFactor
		}
		return inGB * costs.CostPerUnit * 24
	}
	return 0
}

func (e *ExporterApplication) TryAddLine(writer *csv.Writer, data service.DataForDay, clusterId model.ClusterId, metricsKey model.MetricKey) {
	metricData, ok := data.Topics[metricsKey][clusterId]
	if !ok {
		log.Warnf("No data found for cluster %s and metric %s", clusterId, metricsKey)
		return
	}
	costType := ToConfluentCostType(metricsKey)
	costs, err := e.costService.GetCosts(clusterId, costType)
	if err != nil {
		log.Warnf("No cost found for cluster %s and cost type %s", clusterId, costType)
		return
	}

	for topic, m := range metricData {

		err = writer.Write([]string{
			data.DayDate.Format("2006-01-02"),
			fmt.Sprintf("%s-%s-%s", clusterId, topic, metricsKey.ToCsvFormatString()),
			fmt.Sprintf("%f", calcCost(m, costs)),
		})
	}
}

func (e *ExporterApplication) WriteCSV(data service.DataForDay) error {

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

func (e *ExporterApplication) HasExportedDataForDay(date time.Time) bool {

	err := e.EnsureCSVDataFolderExists()
	if err != nil {
		return false
	}

	pathToFile := filepath.Join(CostsExportDir, getFilenameForDay(date))
	_, err = os.Stat(pathToFile)
	return err == nil
}
