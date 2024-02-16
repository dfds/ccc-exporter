package metrics

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
)

type ByCapabilityResponse struct {
	DaysTotal      map[CapabilityId]map[ClusterId]map[MetricKey]float64
	DaysTopicTotal map[CapabilityId]map[ClusterId]map[TopicName]map[MetricKey]float64
}

func ByCapability(allMetrics *AllMetricsResponse) ByCapabilityResponse {
	pattern, err := regexp.Compile("(pub.)?(.*-.{5})\\.")
	if err != nil {
		log.Fatal(err)
	}

	payload := ByCapabilityResponse{}

	daysTotal := make(map[CapabilityId]map[ClusterId]map[MetricKey]float64)
	daysTopicTotal := make(map[CapabilityId]map[ClusterId]map[TopicName]map[MetricKey]float64)

	for metricKey, v := range allMetrics.Days30 {
		for clusterId, vv := range v {
			for topic, value := range vv {
				capabilityRootId := pattern.FindStringSubmatch(topic)

				if len(capabilityRootId) > 2 { // matching pattern of Capability rootid
					// Check that map exists
					if _, ok := daysTotal[CapabilityId(capabilityRootId[2])]; !ok {
						daysTotal[CapabilityId(capabilityRootId[2])] = make(map[ClusterId]map[MetricKey]float64)
					}
					if _, ok := daysTotal[CapabilityId(capabilityRootId[2])][clusterId]; !ok {
						daysTotal[CapabilityId(capabilityRootId[2])][clusterId] = make(map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[CapabilityId(capabilityRootId[2])]; !ok {
						daysTopicTotal[CapabilityId(capabilityRootId[2])] = make(map[ClusterId]map[TopicName]map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId]; !ok {
						daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId] = make(map[TopicName]map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)]; !ok {
						daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)] = make(map[MetricKey]float64)
					}

					// check if key exists
					if _, ok := daysTotal[CapabilityId(capabilityRootId[2])][clusterId][metricKey]; ok {
						daysTotal[CapabilityId(capabilityRootId[2])][clusterId][metricKey] = daysTotal[CapabilityId(capabilityRootId[2])][clusterId][metricKey] + value
					} else {
						daysTotal[CapabilityId(capabilityRootId[2])][clusterId][metricKey] = value
					}
					if _, ok := daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)][metricKey]; ok {
						daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)][metricKey] = daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)][metricKey] + value
					} else {
						daysTopicTotal[CapabilityId(capabilityRootId[2])][clusterId][TopicName(topic)][metricKey] = value
					}

				} else { // everything else
					id := CapabilityId(fmt.Sprintf("unknown-%s", topic))
					if _, ok := daysTotal[id]; !ok {
						daysTotal[id] = make(map[ClusterId]map[MetricKey]float64)
					}
					if _, ok := daysTotal[id][clusterId]; !ok {
						daysTotal[id][clusterId] = make(map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id]; !ok {
						daysTopicTotal[id] = make(map[ClusterId]map[TopicName]map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id][clusterId]; !ok {
						daysTopicTotal[id][clusterId] = make(map[TopicName]map[MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id][clusterId][TopicName(topic)]; !ok {
						daysTopicTotal[id][clusterId][TopicName(topic)] = make(map[MetricKey]float64)
					}

					daysTotal[id][clusterId][metricKey] = value
					daysTopicTotal[id][clusterId][TopicName(topic)][metricKey] = value
				}
			}
		}
	}

	payload.DaysTotal = daysTotal
	payload.DaysTopicTotal = daysTopicTotal

	return payload
}

type CapabilityCostContainer struct {
	Clusters map[ClusterId]*Cluster
}

type Cluster struct {
	Id            string
	MetricsTotal  map[MetricKey]*MetricCost
	MetricsTopics map[TopicName]map[MetricKey]*MetricCost
}

type MetricCost struct {
	MetricValue float64
	CostValue   MetricCostFloat
}

type MetricCostFloat float64

func Float64ToMetricCostFloat(val float64) MetricCostFloat {
	converted, _ := strconv.ParseFloat(fmt.Sprintf("%.6f", val), 64)
	return MetricCostFloat(converted)
}

type CapabilityResponseToCostCsvResponse struct {
	TotalCostByCapability map[CapabilityId]CapabilityCostContainer
	TotalTransferCost     float64
	TotalStorageCost      float64
	TotalStorage          float64
	TotalTransfer         float64
}

func CapabilityResponseToCostCsv(data ByCapabilityResponse, pricingProd Pricing, pricingDev Pricing) CapabilityResponseToCostCsvResponse {
	retentionCostProd := pricingProd.PerBytes().Storage
	retentionCostDev := pricingDev.PerBytes().Storage
	networkTransferProd := pricingProd.PerBytes().NetworkTransfer
	networkTransferDev := pricingDev.PerBytes().NetworkTransfer

	payload := CapabilityResponseToCostCsvResponse{}
	capabilityPayload := map[CapabilityId]CapabilityCostContainer{}

	for capabilityId, clusterMap := range data.DaysTotal {
		capabilityPayload[capabilityId] = CapabilityCostContainer{
			map[ClusterId]*Cluster{},
		}
		for clusterId, metricMap := range clusterMap {
			var retentionCost float64 = 0
			var networkTransferCost float64 = 0
			if clusterId == "lkc-4npj6" {
				retentionCost = retentionCostProd
				networkTransferCost = networkTransferProd
			} else {
				retentionCost = retentionCostDev
				networkTransferCost = networkTransferDev
			}
			capabilityPayload[capabilityId].Clusters[clusterId] = &Cluster{
				Id:            string(clusterId),
				MetricsTotal:  map[MetricKey]*MetricCost{},
				MetricsTopics: map[TopicName]map[MetricKey]*MetricCost{},
			}
			for metricKey, metricValue := range metricMap {
				capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey] = &MetricCost{
					MetricValue: metricValue,
				}

				switch metricKey {
				case ConfluentKafkaServerRetainedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].MetricValue * retentionCost)
					payload.TotalStorageCost = payload.TotalStorageCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue)
					payload.TotalStorage = payload.TotalStorage + (metricValue / 1024 / 1024 / 1024)
				case ConfluentKafkaServerReceivedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].MetricValue * networkTransferCost)
					payload.TotalTransferCost = payload.TotalTransferCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue)
					payload.TotalTransfer = payload.TotalTransfer + (metricValue / 1024 / 1024 / 1024)
				case ConfluentKafkaServerSentBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].MetricValue * networkTransferCost)
					payload.TotalTransferCost = payload.TotalTransferCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue)
					payload.TotalTransfer = payload.TotalTransfer + (metricValue / 1024 / 1024 / 1024)
				default:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = 0.0
				}
			}
		}
	}

	for capabilityId, clusterMap := range data.DaysTopicTotal {
		if _, ok := capabilityPayload[capabilityId]; !ok {
			capabilityPayload[capabilityId] = CapabilityCostContainer{
				map[ClusterId]*Cluster{},
			}
		}

		for clusterId, topicMap := range clusterMap {
			if _, ok := capabilityPayload[capabilityId].Clusters[clusterId]; !ok {
				capabilityPayload[capabilityId].Clusters[clusterId] = &Cluster{
					Id:            string(clusterId),
					MetricsTotal:  map[MetricKey]*MetricCost{},
					MetricsTopics: map[TopicName]map[MetricKey]*MetricCost{},
				}
			}

			for topicName, metricMap := range topicMap {
				capabilityPayload[capabilityId].Clusters[clusterId].MetricsTopics[topicName] = make(map[MetricKey]*MetricCost)
				for metricKey, metricValue := range metricMap {
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTopics[topicName][metricKey] = &MetricCost{
						MetricValue: metricValue,
					}
				}
			}
		}
	}

	payload.TotalCostByCapability = capabilityPayload
	payload.TotalStorageCost = payload.TotalStorageCost * 24 * 30

	fmt.Printf("CapabilityResponseToCostCsv end TotalStorage: %f\n", payload.TotalStorage)

	return payload
}

type Pricing struct {
	NetworkTransfer float64 // flat cost
	Storage         float64 // per hour
}

func (p *Pricing) PerBytes() Pricing {
	return Pricing{
		NetworkTransfer: p.NetworkTransfer / 1024 / 1024 / 1024,
		Storage:         p.Storage / 1024 / 1024 / 1024,
	}
}
