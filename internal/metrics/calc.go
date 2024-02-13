package metrics

import (
	"fmt"
	"log"
	"regexp"
)

func GetTotalCostsByCapability(allMetrics *AllMetricsResponse) {
	pattern, err := regexp.Compile("(pub.)?(.*-.{5})\\.")
	if err != nil {
		log.Fatal(err)
	}

	for metricKey, v := range allMetrics.Days30 {
		fmt.Println(metricKey)
		for clusterId, vv := range v {
			fmt.Printf("  %s\n", clusterId)
			for topic, value := range vv {
				fmt.Printf("    %s: %f\n", topic, value)
				capabilityRootId := pattern.FindStringSubmatch(topic)
				if len(capabilityRootId) > 2 { // matching pattern of Capability rootid
					fmt.Printf("    %s\n", capabilityRootId[2])
				} else { // everything else

				}
			}
		}
	}
}
