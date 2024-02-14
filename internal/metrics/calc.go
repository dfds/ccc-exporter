package metrics

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
)

type ByCapabilityResponse map[CapabilityId]map[ClusterId]map[MetricKey]float64

func ByCapability(allMetrics *AllMetricsResponse) ByCapabilityResponse {
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

	return data
}

type CapabilityCostContainer struct {
	Clusters map[ClusterId]*Cluster
}

type Cluster struct {
	Id      string
	Metrics map[MetricKey]*MetricCost
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
	CostByCapability  map[CapabilityId]CapabilityCostContainer
	TotalTransferCost float64
	TotalStorageCost  float64
	TotalStorage      float64
	TotalTransfer     float64
}

func CapabilityResponseToCostCsv(data ByCapabilityResponse, pricingProd Pricing, pricingDev Pricing) CapabilityResponseToCostCsvResponse {
	retentionCostProd := pricingProd.PerBytes().Storage
	retentionCostDev := pricingDev.PerBytes().Storage
	networkTransferProd := pricingProd.PerBytes().NetworkTransfer
	networkTransferDev := pricingDev.PerBytes().NetworkTransfer

	payload := CapabilityResponseToCostCsvResponse{}
	capabilityPayload := map[CapabilityId]CapabilityCostContainer{}

	for capabilityId, clusterMap := range data {
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
				Id:      string(clusterId),
				Metrics: map[MetricKey]*MetricCost{},
			}
			for metricKey, metricValue := range metricMap {
				capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey] = &MetricCost{
					MetricValue: metricValue,
				}

				switch metricKey {
				case ConfluentKafkaServerRetainedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].MetricValue * retentionCost)
					payload.TotalStorageCost = payload.TotalStorageCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue)
					payload.TotalStorage = payload.TotalStorage + (metricValue / 1024 / 1024 / 1024)
				case ConfluentKafkaServerReceivedBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].MetricValue * networkTransferCost)
					payload.TotalTransferCost = payload.TotalTransferCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue)
					payload.TotalTransfer = payload.TotalTransfer + (metricValue / 1024 / 1024 / 1024)
				case ConfluentKafkaServerSentBytes:
					capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue = Float64ToMetricCostFloat(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].MetricValue * networkTransferCost)
					payload.TotalTransferCost = payload.TotalTransferCost + float64(capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue)
					payload.TotalTransfer = payload.TotalTransfer + (metricValue / 1024 / 1024 / 1024)
				default:
					capabilityPayload[capabilityId].Clusters[clusterId].Metrics[metricKey].CostValue = 0.0
				}
			}
		}
	}

	payload.CostByCapability = capabilityPayload
	payload.TotalStorageCost = payload.TotalStorageCost * 24 * 30

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
