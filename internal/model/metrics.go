package model

import "go.dfds.cloud/ccc-exporter/internal/service"

type MetricKey string

const (
	ConfluentKafkaServerReceivedBytes MetricKey = "confluent_kafka_server_received_bytes"
	ConfluentKafkaServerSentBytes     MetricKey = "confluent_kafka_server_sent_bytes"
	ConfluentKafkaServerRetainedBytes MetricKey = "confluent_kafka_server_retained_bytes"
)

var ConfluentMetrics = []MetricKey{ConfluentKafkaServerReceivedBytes, ConfluentKafkaServerSentBytes, ConfluentKafkaServerRetainedBytes}

func (m MetricKey) IsValid() bool {
	switch m {
	case ConfluentKafkaServerReceivedBytes, ConfluentKafkaServerSentBytes, ConfluentKafkaServerRetainedBytes:
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
	}
	return "INVALID"
}

func (m MetricKey) ToConfluentCostType() service.CostType {
	switch m {
	case ConfluentKafkaServerReceivedBytes:
		return service.CostTypeKafkaNetworkRead
	case ConfluentKafkaServerSentBytes:
		return service.CostTypeKafkaNetworkWrite
	case ConfluentKafkaServerRetainedBytes:
		return service.CostTypeKafkaStorage
	}
	return "INVALID"
}
