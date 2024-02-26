package service

import (
	"fmt"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"time"
)

type CostType string

const (
	CostTypeConnectCapable    CostType = "CONNECT_CAPACITY"
	CostTypeConnectNumTasks   CostType = "CONNECT_NUM_TASKS"
	CostTypeConnectThroughPut CostType = "CONNECT_THROUGHPUT"
	CostTypeKafkaBaseCost     CostType = "KAFKA_BASE_COST"
	CostTypeKafkaStorage      CostType = "KAFKA_STORAGE"
	CostTypeKafkaNetworkRead  CostType = "KAFKA_NETWORK_READ"
	CostTypeKafkaNetworkWrite CostType = "KAFKA_NETWORK_WRITE"
	CostTypeKafkaNumCkus      CostType = "KAFKA_NUM_CKUS"
	CostTypeKafkaPartition    CostType = "KAFKA_PARTITION"
)

type ProductType string

const (
	ProductTypeConnect ProductType = "CONNECT"
	ProductTypeKafka   ProductType = "KAFKA"
	ProductTypeSupport ProductType = "SUPPORT_CLOUD_BUSINESS"
)

type CostUnit string

const (
	USDPerGB     = "USD/GB"
	USDPerGBHour = "USD/GB-hour"
	Other        = "Other"
)

type ConfluentCost struct {
	CostType    CostType
	ProductType ProductType
	ClusterId   model.ClusterId
	CostPerUnit float64
	CostUnit    CostUnit
	TotalCost   float64
}
type ConfluentCostService struct {
	// Can change from day to day, start with just keeping the latest
	cachedCosts map[model.ClusterId]map[CostType]ConfluentCost
}

func (c *ConfluentCostService) SetupTestCosts() {
	for _, cluster := range model.ConfluentClusters {
		c.cachedCosts[cluster] = make(map[CostType]ConfluentCost)
		c.cachedCosts[cluster][CostTypeKafkaNetworkWrite] = ConfluentCost{
			CostType:    CostTypeKafkaNetworkWrite,
			ProductType: ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.066,
			CostUnit:    USDPerGB,
			TotalCost:   1.23,
		}

		c.cachedCosts[cluster][CostTypeKafkaNetworkRead] = ConfluentCost{
			CostType:    CostTypeKafkaNetworkRead,
			ProductType: ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.1265,
			CostUnit:    USDPerGB,
			TotalCost:   6.0624,
		}

		c.cachedCosts[cluster][CostTypeKafkaStorage] = ConfluentCost{
			CostType:    CostTypeKafkaStorage,
			ProductType: ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.00012055,
			TotalCost:   0.0181,
			CostUnit:    USDPerGBHour,
		}
	}
}

func NewConfluentCostService(useTestCosts bool) *ConfluentCostService {
	manager := &ConfluentCostService{cachedCosts: make(map[model.ClusterId]map[CostType]ConfluentCost)}

	if useTestCosts {
		manager.SetupTestCosts()
	}
	return manager
}

func (c *ConfluentCostService) GetCosts(clusterId model.ClusterId, costType CostType) (ConfluentCost, error) {
	if costs, ok := c.cachedCosts[clusterId]; ok {
		if cost, ok := costs[costType]; ok {
			return cost, nil
		}
	}
	return ConfluentCost{}, fmt.Errorf("no costs found for cluster %s", clusterId)
}

func (c *ConfluentCostService) HasCostsForDate(date time.Time) bool {
	// TODO: Add cached costs per dates
	// for now we just check if any costs are cached
	_, err := c.GetCosts(model.ClusterIdProd, CostTypeKafkaNetworkWrite)
	if err != nil {
		return false
	}
	return true

}
