package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
)

type DataF64 struct {
	Key   string
	Value float64
}

func GetTotalCostsByCapability(allMetrics *AllMetricsResponse) {
	pattern, err := regexp.Compile("(pub.)?(.*-.{5})\\.")
	if err != nil {
		log.Fatal(err)
	}

	data := make(map[CapabilityId]map[ClusterId]map[MetricKey]float64)

	for metricKey, v := range allMetrics.Days30 {
		for clusterId, vv := range v {
			for topic, value := range vv {
				capabilityRootId := pattern.FindStringSubmatch(topic)
				if len(capabilityRootId) > 2 { // matching pattern of Capability rootid
					// Check that map exists
					if _, ok := data[CapabilityId(capabilityRootId[2])]; !ok {
						data[CapabilityId(capabilityRootId[2])] = make(map[ClusterId]map[MetricKey]float64)
					}
					if _, ok := data[CapabilityId(capabilityRootId[2])][clusterId]; !ok {
						data[CapabilityId(capabilityRootId[2])][clusterId] = make(map[MetricKey]float64)
					}

					data[CapabilityId(capabilityRootId[2])][clusterId][metricKey] = value
				} else { // everything else
					id := CapabilityId(fmt.Sprintf("unknown-%s", topic))
					if _, ok := data[id]; !ok {
						data[id] = make(map[ClusterId]map[MetricKey]float64)
					}
					if _, ok := data[id][clusterId]; !ok {
						data[id][clusterId] = make(map[MetricKey]float64)
					}
					data[id][clusterId][metricKey] = value
				}
			}
		}
	}

	serialised, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(string(serialised))

}
