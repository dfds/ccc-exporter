package metrics

import (
	"fmt"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"log"
	"time"
)

type Gatherer struct {
	client *client.Client
}

func NewGatherer(client *client.Client) *Gatherer {
	return &Gatherer{client: client}
}

func (g *Gatherer) GetAllMetrics() {
	baseQuery := "sum_over_time(confluent_kafka_server_retained_bytes[1d]"
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
			if vector.Value.Value != "0" {
				fmt.Println(vector)
			}
		}
	}
}
