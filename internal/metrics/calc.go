package metrics

import (
	"fmt"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/service"
	"log"
	"regexp"
	"strconv"
)

type ByCapabilityResponse struct {
	DaysTotal      map[model.CapabilityId]map[model.ClusterId]map[model.MetricKey]float64
	DaysTopicTotal map[model.CapabilityId]map[model.ClusterId]map[model.TopicName]map[model.MetricKey]float64
}

func ByCapability(allMetrics *service.AllMetricsResponse) ByCapabilityResponse {
	pattern, err := regexp.Compile("(pub.)?(.*-.{5})\\.")
	if err != nil {
		log.Fatal(err)
	}

	payload := ByCapabilityResponse{}

	daysTotal := make(map[model.CapabilityId]map[model.ClusterId]map[model.MetricKey]float64)
	daysTopicTotal := make(map[model.CapabilityId]map[model.ClusterId]map[model.TopicName]map[model.MetricKey]float64)

	for metricKey, v := range allMetrics.Days30 {
		for clusterId, vv := range v {
			for topic, value := range vv {
				capabilityRootId := pattern.FindStringSubmatch(topic)

				if len(capabilityRootId) > 2 { // matching pattern of Capability rootid
					// Check that map exists
					if _, ok := daysTotal[model.CapabilityId(capabilityRootId[2])]; !ok {
						daysTotal[model.CapabilityId(capabilityRootId[2])] = make(map[model.ClusterId]map[model.MetricKey]float64)
					}
					if _, ok := daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId]; !ok {
						daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId] = make(map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[model.CapabilityId(capabilityRootId[2])]; !ok {
						daysTopicTotal[model.CapabilityId(capabilityRootId[2])] = make(map[model.ClusterId]map[model.TopicName]map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId]; !ok {
						daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId] = make(map[model.TopicName]map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)]; !ok {
						daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)] = make(map[model.MetricKey]float64)
					}

					// check if key exists
					if _, ok := daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId][metricKey]; ok {
						daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId][metricKey] = daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId][metricKey] + value
					} else {
						daysTotal[model.CapabilityId(capabilityRootId[2])][clusterId][metricKey] = value
					}
					if _, ok := daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)][metricKey]; ok {
						daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)][metricKey] = daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)][metricKey] + value
					} else {
						daysTopicTotal[model.CapabilityId(capabilityRootId[2])][clusterId][model.TopicName(topic)][metricKey] = value
					}

				} else { // everything else
					id := model.CapabilityId(fmt.Sprintf("unknown-%s", topic))
					if _, ok := daysTotal[id]; !ok {
						daysTotal[id] = make(map[model.ClusterId]map[model.MetricKey]float64)
					}
					if _, ok := daysTotal[id][clusterId]; !ok {
						daysTotal[id][clusterId] = make(map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id]; !ok {
						daysTopicTotal[id] = make(map[model.ClusterId]map[model.TopicName]map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id][clusterId]; !ok {
						daysTopicTotal[id][clusterId] = make(map[model.TopicName]map[model.MetricKey]float64)
					}
					if _, ok := daysTopicTotal[id][clusterId][model.TopicName(topic)]; !ok {
						daysTopicTotal[id][clusterId][model.TopicName(topic)] = make(map[model.MetricKey]float64)
					}

					daysTotal[id][clusterId][metricKey] = value
					daysTopicTotal[id][clusterId][model.TopicName(topic)][metricKey] = value
				}
			}
		}
	}

	payload.DaysTotal = daysTotal
	payload.DaysTopicTotal = daysTopicTotal

	return payload
}

type CapabilityCostContainer struct {
	Clusters map[model.ClusterId]*Cluster
}

type Cluster struct {
	Id            string
	MetricsTotal  map[model.MetricKey]*MetricCost
	MetricsTopics map[model.TopicName]map[model.MetricKey]*MetricCost
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
	TotalCostByCapability map[model.CapabilityId]CapabilityCostContainer
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
	capabilityPayload := map[model.CapabilityId]CapabilityCostContainer{}

	for capabilityId, clusterMap := range data.DaysTotal {
		capabilityPayload[capabilityId] = CapabilityCostContainer{
			map[model.ClusterId]*Cluster{},
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
				MetricsTotal:  map[model.MetricKey]*MetricCost{},
				MetricsTopics: map[model.TopicName]map[model.MetricKey]*MetricCost{},
			}
			for metricKey, metricValue := range metricMap {
				capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey] = &MetricCost{
					MetricValue: metricValue,
				}

				switch metricKey {
				case model.ConfluentKafkaServerRetainedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].MetricValue * retentionCost)
					payload.TotalStorageCost = payload.TotalStorageCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue)
					payload.TotalStorage = payload.TotalStorage + (metricValue / 1024 / 1024 / 1024)
				case model.ConfluentKafkaServerReceivedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].MetricValue * networkTransferCost)
					payload.TotalTransferCost = payload.TotalTransferCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].MetricsTotal[metricKey].CostValue)
					payload.TotalTransfer = payload.TotalTransfer + (metricValue / 1024 / 1024 / 1024)
				case model.ConfluentKafkaServerSentBytes:
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
				map[model.ClusterId]*Cluster{},
			}
		}

		for clusterId, topicMap := range clusterMap {
			if _, ok := capabilityPayload[capabilityId].Clusters[clusterId]; !ok {
				capabilityPayload[capabilityId].Clusters[clusterId] = &Cluster{
					Id:            string(clusterId),
					MetricsTotal:  map[model.MetricKey]*MetricCost{},
					MetricsTopics: map[model.TopicName]map[model.MetricKey]*MetricCost{},
				}
			}

			for topicName, metricMap := range topicMap {
				capabilityPayload[capabilityId].Clusters[clusterId].MetricsTopics[topicName] = make(map[model.MetricKey]*MetricCost)
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
