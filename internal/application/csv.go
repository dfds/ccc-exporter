package application

import (
	"fmt"
	"github.com/gofiber/fiber/v2/log"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/util"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)
import "encoding/csv"

const CostsExportDir = "export"
const UnknownPlaceholder = "UNKNOWN"

func (e *ExporterApplication) EnsureCSVDataFolderExists() error {
	err := os.MkdirAll(CostsExportDir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func calcCost(m model.MetricData, costs model.KafkaConfluentCost) float64 {
	inGB := m.Value / 1024 / 1024 / 1024
	fmt.Printf("%s\n", costs.CostType)
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

func (e *ExporterApplication) TryAddLine(writer *csv.Writer, data model.MetricsDataForDay, clusterId model.ClusterId, pattern *regexp.Regexp, metricsKey model.MetricKey) {
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
		capabilityRootId := pattern.FindStringSubmatch(string(topic))
		capability := "UNKNOWN"

		if len(capabilityRootId) > 2 { // matching pattern of Capability rootid
			capability = capabilityRootId[2]
			if strings.Contains(capabilityRootId[2], "_confluent-ksql") {
				capability = UnknownPlaceholder
			}
		}

		err = writer.Write([]string{
			data.DayDate.ToCSVString(),
			fmt.Sprintf("%f", calcCost(m, costs)),
			string(topic),
			string(clusterId),
			metricsKey.ToCsvFormatString(),
			capability,
		})

		//key := fmt.Sprintf("%s-%s-%s", clusterId, topic, metricsKey.ToCsvFormatString())
		//cost := calcCost(m, costs)
		//if strings.Contains(key, "lkc-4npj6-dataplatform-ajamn.ft-unithandling-phx-source-written-bytes") || strings.Contains(key, "lkc-4npj6-onboardcustomers-npxkm.customerbookings-internal-prod-written-bytes") {
		//	fmt.Printf("key: %s\ncost: %f\n", key, cost)
		//}
	}
}

func (e *ExporterApplication) ReadCsvRaw(date util.YearMonthDayDate) ([]byte, error) {
	pathToFile := filepath.Join(CostsExportDir, date.ToFileNameFormat())
	byteData, err := os.ReadFile(pathToFile)
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

	pathToFile := filepath.Join(CostsExportDir, data.DayDate.ToFileNameFormat())
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

	pattern, err := regexp.Compile("(pub.)?(.*-.{5})\\.")
	if err != nil {
		return err
	}

	// new headers: Date,Cost,Name,Action,Capability
	//headers := []string{"Date", "ServiceName", "Cost"}
	headers := []string{"Date", "Cost", "Name", "ClusterId", "Action", "Capability"}
	err = writer.Write(headers)
	if err != nil {
		return err
	}
	for _, clusterId := range model.ConfluentClusters {
		e.TryAddLine(writer, data, clusterId, pattern, model.ConfluentKafkaServerReceivedBytes)
		e.TryAddLine(writer, data, clusterId, pattern, model.ConfluentKafkaServerSentBytes)
		e.TryAddLine(writer, data, clusterId, pattern, model.ConfluentKafkaServerRetainedBytes)
	}

	return nil
}

func (e *ExporterApplication) HasExportedDataForDay(date util.YearMonthDayDate) bool {

	err := e.EnsureCSVDataFolderExists()
	if err != nil {
		return false
	}

	pathToFile := filepath.Join(CostsExportDir, date.ToFileNameFormat())
	_, err = os.Stat(pathToFile)
	return err == nil
}
