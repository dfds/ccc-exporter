package model

import "go.dfds.cloud/ccc-exporter/internal/util"

type MetricKey string

const (
	ConfluentKafkaServerReceivedBytes MetricKey = "confluent_kafka_server_received_bytes"
	ConfluentKafkaServerSentBytes     MetricKey = "confluent_kafka_server_sent_bytes"
	ConfluentKafkaServerRetainedBytes MetricKey = "confluent_kafka_server_retained_bytes"
	ConfluentKafkaServerResponseBytes MetricKey = "confluent_kafka_server_response_bytes"
	ConfluentKafkaServerRequestBytes  MetricKey = "confluent_kafka_server_request_bytes"
)

var ConfluentMetrics = []MetricKey{
	ConfluentKafkaServerReceivedBytes,
	ConfluentKafkaServerSentBytes,
	ConfluentKafkaServerRetainedBytes,
	ConfluentKafkaServerResponseBytes,
	ConfluentKafkaServerRequestBytes}

func (m MetricKey) IsValid() bool {
	switch m {
	case ConfluentKafkaServerReceivedBytes, ConfluentKafkaServerSentBytes, ConfluentKafkaServerRetainedBytes, ConfluentKafkaServerResponseBytes, ConfluentKafkaServerRequestBytes:
		return true
	}
	return false
}
func (m MetricKey) ToCsvFormatString() string {
	switch m {
	case ConfluentKafkaServerReceivedBytes:
		return "read-bytes"
	case ConfluentKafkaServerSentBytes:
		return "written-bytes"
	case ConfluentKafkaServerRetainedBytes:
		return "stored-bytes"
	case ConfluentKafkaServerResponseBytes:
		return "response-bytes"
	case ConfluentKafkaServerRequestBytes:
		return "request-bytes"
	}
	return "INVALID"
}
func (m MetricKey) ToConfluentCostType() CostType {
	switch m {
	case ConfluentKafkaServerReceivedBytes:
		return CostTypeKafkaNetworkRead
	case ConfluentKafkaServerSentBytes:
		return CostTypeKafkaNetworkWrite
	case ConfluentKafkaServerRetainedBytes:
		return CostTypeKafkaStorage
	}
	return "INVALID"
}

type MetricsDataForDay struct {
	DayDate util.YearMonthDayDate
	Topics  map[MetricKey]map[ClusterId]map[TopicName]MetricData

	TotalCostPerClusterWrittenBytes map[ClusterId]float64
	TotalCostPerClusterReadBytes    map[ClusterId]float64

	TotalCostWrittenBytes float64
	TotalCostReadBytes    float64
}

type MetricData struct {
	Time  float64
	Value float64
}
