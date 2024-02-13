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

func (g *Gatherer) GetAllMetrics() map[MetricKey]map[string]float64 {
	dataStore30Days := make(map[MetricKey]map[string]float64)

	for _, metricKey := range ConfluentMetrics {
		// check if metricKey exists in dataStore30Days, if not, init
		if _, ok := dataStore30Days[metricKey]; !ok {
			dataStore30Days[metricKey] = make(map[string]float64)
		}

		baseQuery := fmt.Sprintf("sum_over_time(%s[1d]", metricKey)
		for i := 0; i <= 30; i++ {
			var query string = baseQuery
			if i != 0 {
				query = fmt.Sprintf("%s offset %dd)", baseQuery, i)
			} else {
				query = fmt.Sprintf("%s)", query)
			}

			fmt.Println(query)

			queryResp, err := g.client.Query(query, float64(time.Now().Unix()))
			if err != nil {
				log.Fatal(err)
			}

			data, err := client.ResultToVector(queryResp.Data.Result)
			if err != nil {
				log.Fatal(err)
			}

			for _, vector := range data {
				f64, _ := strconv.ParseFloat(vector.Value.Value, 64)
				dataStore30Days[metricKey][vector.Metric.Topic] = dataStore30Days[metricKey][vector.Metric.Topic] + f64
			}
		}
	}

	return dataStore30Days
}
