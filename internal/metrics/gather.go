package metrics

import (
	"fmt"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"log"
	"strconv"
	"time"
)

type Gatherer struct {
	client *client.Client
}

func NewGatherer(client *client.Client) *Gatherer {
	return &Gatherer{client: client}
}

type MetricKey string
type ClusterId string

type MetricData struct {
	Time  float64
	Value float64
}

type AllMetricsResponse struct {
	Days30 map[MetricKey]map[ClusterId]map[string]float64
	PerDay map[MetricKey]map[ClusterId]map[string][]MetricData
}

func (g *Gatherer) GetAllMetrics() *AllMetricsResponse {
	dataStore30Days := make(map[MetricKey]map[ClusterId]map[string]float64)
	dataStorePerDay := make(map[MetricKey]map[ClusterId]map[string][]MetricData)
	now := time.Now()

	for _, metricKey := range ConfluentMetrics {
		// check if metricKey exists in dataStore30Days and dataStorePerDay, if not, init
		if _, ok := dataStore30Days[metricKey]; !ok {
			dataStore30Days[metricKey] = make(map[ClusterId]map[string]float64)
		}
		if _, ok := dataStorePerDay[metricKey]; !ok {
			dataStorePerDay[metricKey] = make(map[ClusterId]map[string][]MetricData)
		}

		baseQuery := fmt.Sprintf("sum_over_time(%s[1d]", metricKey)
		for i := 0; i <= 30; i++ {
			var query string = baseQuery
			timestamp := now
			if i != 0 {
				query = fmt.Sprintf("%s offset %dd)", baseQuery, i)
				timestamp = now.Add(time.Duration(i*24) * -time.Hour)
			} else {
				query = fmt.Sprintf("%s)", query)
			}

			fmt.Println(query)

			queryResp, err := g.client.Query(query, float64(now.Unix()))
			if err != nil {
				log.Fatal(err)
			}

			data, err := client.ResultToVector(queryResp.Data.Result)
			if err != nil {
				log.Fatal(err)
			}

			for _, vector := range data {
				if _, ok := dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)]; !ok {
					dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)] = map[string][]MetricData{}
				}
				if _, ok := dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic]; !ok {
					dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = []MetricData{}
				}
				if _, ok := dataStore30Days[metricKey][ClusterId(vector.Metric.KafkaID)]; !ok {
					dataStore30Days[metricKey][ClusterId(vector.Metric.KafkaID)] = map[string]float64{}
				}

				f64, _ := strconv.ParseFloat(vector.Value.Value, 64)

				dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = append(dataStorePerDay[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic], MetricData{
					Time:  float64(timestamp.Unix()),
					Value: f64,
				})
				dataStore30Days[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = dataStore30Days[metricKey][ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] + f64
			}
		}
	}

	return &AllMetricsResponse{
		Days30: dataStore30Days,
		PerDay: dataStorePerDay,
	}
}
