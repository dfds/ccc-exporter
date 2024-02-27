package service

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"os"
	"time"
)

type ConfluentCostService struct {
	// Can change from day to day, start with just keeping the latest
	cachedCosts map[model.ClusterId]map[model.CostType]model.ConfluentCost
}

func (c *ConfluentCostService) CacheCosts(costs model.ConfluentCostResponse) {

	for _, cluster := range model.ConfluentClusters {
		c.cachedCosts[cluster] = make(map[model.CostType]model.ConfluentCost)
	}

	for _, cost := range costs.Data {
		costType, err := model.TryParseCostType(cost.LineType)
		if err != nil {
			log.Error().Msgf("failed to parse cost type: %s", err)
			continue
		}
		costUnit, err := model.TryParseCostUnit(cost.Unit)
		if err != nil && costType != model.CostTypeSupport {
			log.Error().Msgf("failed to parse cost unit: %s", err)
			continue
		}
		productType, err := model.TryParseProductType(cost.Product)
		if err != nil {
			log.Error().Msgf("failed to parse product type: %s", err)
			continue
		}
		switch productType {
		case model.ProductTypeConnect:
		case model.ProductTypeKafka:
			clusterId, err := model.TryParseClusterId(cost.Resource.Id)
			if err != nil {
				log.Error().Msgf("failed to parse cluster id: %s", err)
				continue
			}
			c.cachedCosts[clusterId][costType] = model.ConfluentCost{
				CostType:    costType,
				ProductType: productType,
				ClusterId:   clusterId,
				CostPerUnit: cost.Price,
				CostUnit:    costUnit,
				TotalCost:   cost.Amount,
			}
		case model.ProductTypeSupport:
		}
	}
}

func (c *ConfluentCostService) SetupTestCostsFromFile() bool {
	byteData, err := os.ReadFile("data.json")
	if err != nil {
		return false
	}

	var costs model.ConfluentCostResponse
	err = json.Unmarshal(byteData, &costs)
	if err != nil {
		log.Err(err).Msgf("failed to unmarshal data.json")
		return false
	}

	c.CacheCosts(costs)

	return true
}

func (c *ConfluentCostService) SetupHardcodedTestCosts() {
	for _, cluster := range model.ConfluentClusters {
		c.cachedCosts[cluster] = make(map[model.CostType]model.ConfluentCost)
		c.cachedCosts[cluster][model.CostTypeKafkaNetworkWrite] = model.ConfluentCost{
			CostType:    model.CostTypeKafkaNetworkWrite,
			ProductType: model.ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.066,
			CostUnit:    model.GB,
			TotalCost:   1.23,
		}

		c.cachedCosts[cluster][model.CostTypeKafkaNetworkRead] = model.ConfluentCost{
			CostType:    model.CostTypeKafkaNetworkRead,
			ProductType: model.ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.1265,
			CostUnit:    model.GB,
			TotalCost:   6.0624,
		}

		c.cachedCosts[cluster][model.CostTypeKafkaStorage] = model.ConfluentCost{
			CostType:    model.CostTypeKafkaStorage,
			ProductType: model.ProductTypeKafka,
			ClusterId:   cluster,
			CostPerUnit: 0.00012055,
			TotalCost:   0.0181,
			CostUnit:    model.GBHour,
		}
	}
}

func NewConfluentCostService(useTestCosts bool) *ConfluentCostService {
	manager := &ConfluentCostService{cachedCosts: make(map[model.ClusterId]map[model.CostType]model.ConfluentCost)}

	if useTestCosts {
		log.Info().Msgf("Using test costs")

		if manager.SetupTestCostsFromFile() {
			log.Info().Msgf("data.json found, using cached costs")
			return manager
		}
		log.Info().Msgf("data.json not found, using hardcoded test costs")
		manager.SetupHardcodedTestCosts()
	}
	return manager
}

func (c *ConfluentCostService) GetCosts(clusterId model.ClusterId, costType model.CostType) (model.ConfluentCost, error) {
	if costs, ok := c.cachedCosts[clusterId]; ok {
		if cost, ok := costs[costType]; ok {
			return cost, nil
		}
	}
	return model.ConfluentCost{}, fmt.Errorf("no costs found for cluster %s", clusterId)
}

func (c *ConfluentCostService) HasCostsForDate(date time.Time) bool {
	// TODO: Add cached costs per dates
	// for now we just check if any costs are cached
	_, err := c.GetCosts(model.ClusterIdProd, model.CostTypeKafkaNetworkWrite)
	if err != nil {
		return false
	}
	return true

}
