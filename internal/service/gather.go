package service

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/utils"
	"strconv"
	"time"
)

type GathererService struct {
	client      *client.PrometheusClient
	cachedUsage map[utils.YearMonthDayDate]model.MetricsDataForDay
}

func NewGatherer(client *client.PrometheusClient) *GathererService {
	return &GathererService{client: client,
		cachedUsage: make(map[utils.YearMonthDayDate]model.MetricsDataForDay)}
}

type AllMetricsResponse struct {
	Days30 map[model.MetricKey]map[model.ClusterId]map[string]float64
	PerDay map[model.MetricKey]map[model.ClusterId]map[string][]model.MetricData
}

func getQueryForMetric(metricKey model.MetricKey, timeDiffInSeconds int) string {
	innerQuery := fmt.Sprintf("%s[1d] offset %ds", metricKey, timeDiffInSeconds)
	if metricKey == model.ConfluentKafkaServerRetainedBytes {
		return innerQuery
	}
	return fmt.Sprintf("sum_over_time(%s)", innerQuery)
}

func (g *GathererService) GetMetricsForDay(targetTime utils.YearMonthDayDate) (model.MetricsDataForDay, error) {

	now := time.Now().UTC()

	cached, ok := g.cachedUsage[targetTime]
	if ok {
		return cached, nil
	}

	timeDiffInSeconds := int(now.Sub(targetTime.ToTimeUTC()).Seconds()) // loss of precision, but good enough for this use case
	if timeDiffInSeconds <= 0 {
		return model.MetricsDataForDay{}, fmt.Errorf("cannot get metrics for current/future day")
	}

	metricsForDayAndTopic := make(map[model.MetricKey]map[model.ClusterId]map[model.TopicName]model.MetricData)
	for _, metric := range model.ConfluentMetrics {
		metricsForDayAndTopic[metric] = make(map[model.ClusterId]map[model.TopicName]model.MetricData)
		for _, clusterId := range model.ConfluentClusters {
			metricsForDayAndTopic[metric][clusterId] = make(map[model.TopicName]model.MetricData)
		}
	}

	for _, metricKey := range model.ConfluentMetrics {
		query := getQueryForMetric(metricKey, timeDiffInSeconds)
		queryResp, err := g.client.Query(query, float64(now.Unix()))
		if err != nil {
			return model.MetricsDataForDay{}, err
		}

		data, err := client.ResultToVector(queryResp.Data.Result)
		if err != nil {
			return model.MetricsDataForDay{}, err
		}
		for _, vector := range data {
			clusterId, err := model.TryParseClusterId(vector.Metric.KafkaID)

			if err != nil {
				log.Err(err).Msgf("error when attempting to parse KafkaId returned from prometheus")
				continue
			}

			valueAsFloat, err := strconv.ParseFloat(vector.Value.Value, 64)
			if err != nil {
				log.Err(err).Msgf("error when attempting to parse value returned from prometheus")
				continue
			}
			topicName := model.TopicName(vector.Metric.Topic)
			if _, ok := metricsForDayAndTopic[metricKey][clusterId][topicName]; ok {
				log.Fatal().Msgf("duplicate metric found for topic: %s", vector.Metric.Topic)
			}
			metricsForDayAndTopic[metricKey][clusterId][topicName] = model.MetricData{
				Time:  vector.Value.Time,
				Value: valueAsFloat,
			}
		}
	}

	g.cachedUsage[targetTime] = model.MetricsDataForDay{
		DayDate: targetTime,
		Topics:  metricsForDayAndTopic,
	}

	return g.cachedUsage[targetTime], nil
}

func (g *GathererService) GetAllMetrics() *AllMetricsResponse {
	dataStore30Days := make(map[model.MetricKey]map[model.ClusterId]map[string]float64)
	dataStorePerDay := make(map[model.MetricKey]map[model.ClusterId]map[string][]model.MetricData)
	now := time.Now()

	for _, metricKey := range model.ConfluentMetrics {
		// check if metricKey exists in dataStore30Days and dataStorePerDay, if not, init
		if _, ok := dataStore30Days[metricKey]; !ok {
			dataStore30Days[metricKey] = make(map[model.ClusterId]map[string]float64)
		}
		if _, ok := dataStorePerDay[metricKey]; !ok {
			dataStorePerDay[metricKey] = make(map[model.ClusterId]map[string][]model.MetricData)
		}

		baseQuery := fmt.Sprintf("sum_over_time(%s[1d]", metricKey)
		for i := 0; i <= 30; i++ {
			var query = baseQuery
			timestamp := now
			if i != 0 {
				query = fmt.Sprintf("%s offset %dd)", baseQuery, i)
				timestamp = now.Add(time.Duration(i*24) * -time.Hour)
				if metricKey == model.ConfluentKafkaServerRetainedBytes {
					query = fmt.Sprintf("%s offset %dd", model.ConfluentKafkaServerRetainedBytes, i)
				}
			} else {
				query = fmt.Sprintf("%s)", query)
				if metricKey == model.ConfluentKafkaServerRetainedBytes {
					query = fmt.Sprintf("%s offset 1h", model.ConfluentKafkaServerRetainedBytes) // not perfect, WIP
				}
			}

			fmt.Println(query)

			queryResp, err := g.client.Query(query, float64(now.Unix()))
			if err != nil {
				log.Fatal().Err(err).Msg("error querying prometheus")
			}

			data, err := client.ResultToVector(queryResp.Data.Result)
			if err != nil {
				log.Fatal().Err(err).Msg("error parsing prometheus response")
			}

			for _, vector := range data {
				if _, ok := dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)]; !ok {
					dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)] = map[string][]model.MetricData{}
				}
				if _, ok := dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic]; !ok {
					dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = []model.MetricData{}
				}
				if _, ok := dataStore30Days[metricKey][model.ClusterId(vector.Metric.KafkaID)]; !ok {
					dataStore30Days[metricKey][model.ClusterId(vector.Metric.KafkaID)] = map[string]float64{}
				}

				f64, _ := strconv.ParseFloat(vector.Value.Value, 64)

				dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = append(dataStorePerDay[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic], model.MetricData{
					Time:  float64(timestamp.Unix()),
					Value: f64,
				})

				if metricKey == model.ConfluentKafkaServerRetainedBytes {
					if i == 0 {
						dataStore30Days[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = f64
					}
				} else {
					dataStore30Days[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] = dataStore30Days[metricKey][model.ClusterId(vector.Metric.KafkaID)][vector.Metric.Topic] + f64
				}

			}
		}
	}

	return &AllMetricsResponse{
		Days30: dataStore30Days,
		PerDay: dataStorePerDay,
	}
}
