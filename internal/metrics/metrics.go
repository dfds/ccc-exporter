package metrics

const (
	ConfluentKafkaServerReceivedBytes = "confluent_kafka_server_received_bytes"
	ConfluentKafkaServerSentBytes     = "confluent_kafka_server_sent_bytes"
	ConfluentKafkaServerRetainedBytes = "confluent_kafka_server_retained_bytes"
)

var ConfluentMetrics = []MetricKey{ConfluentKafkaServerReceivedBytes, ConfluentKafkaServerSentBytes, ConfluentKafkaServerRetainedBytes}
