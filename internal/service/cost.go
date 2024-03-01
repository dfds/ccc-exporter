package service

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.dfds.cloud/ccc-exporter/internal/client"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"go.dfds.cloud/ccc-exporter/internal/util"
	"os"
	"time"
)

type confluentCostForDay struct {
	kafka map[model.ClusterId]map[model.CostType]model.KafkaConfluentCost
}

func newConfluentCostForDay() *confluentCostForDay {
	return &confluentCostForDay{
		kafka: make(map[model.ClusterId]map[model.CostType]model.KafkaConfluentCost),
	}
}
func (c *confluentCostForDay) setupClusters() {
	for _, cluster := range model.ConfluentClusters {
		c.kafka[cluster] = make(map[model.CostType]model.KafkaConfluentCost)
	}
}

type ConfluentCostService struct {
	// Can change from day to day, start with just keeping the latest

	cachedCosts          map[util.YearMonthDayDate]confluentCostForDay
	confluentCloudClient *client.ConfluentCloudClient
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

	// try to deduce the date from the data
	var date time.Time
	for _, cost := range costs.Data {
		if cost.StartDate == "" {
			continue
		}
		date, err = time.Parse("2006-01-02", cost.StartDate)
		if err != nil {
			log.Err(err).Msgf("failed to parse date from data.json")
			return false
		}
		break
	}
	if date.IsZero() {
		log.Error().Msgf("no date found in data.json")
		return false
	}

	c.CacheCosts(util.ToYearMonthDayDate(date), costs)

	return true
}
func NewConfluentCostService(confluentCloudClient *client.ConfluentCloudClient, useTestCosts bool) *ConfluentCostService {
	manager := &ConfluentCostService{
		cachedCosts:          make(map[util.YearMonthDayDate]confluentCostForDay),
		confluentCloudClient: confluentCloudClient,
	}

	if useTestCosts {
		log.Info().Msgf("Using test costs")

		if manager.SetupTestCostsFromFile() {
			log.Info().Msgf("data.json found, using cached costs")
			return manager
		}
		log.Fatal().Msgf("data.json not found!")

	}
	return manager
}

func (c *ConfluentCostService) CacheCosts(date util.YearMonthDayDate, costs model.ConfluentCostResponse) {

	if _, ok := c.cachedCosts[date]; !ok {
		newCosts := newConfluentCostForDay()
		newCosts.setupClusters()
		c.cachedCosts[date] = *newCosts
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
			c.cachedCosts[date].kafka[clusterId][costType] = model.KafkaConfluentCost{
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

func (c *ConfluentCostService) GetKafkaCosts(date util.YearMonthDayDate, clusterId model.ClusterId, costType model.CostType) (model.KafkaConfluentCost, error) {

	costsForDay, ok := c.cachedCosts[date]
	if !ok {
		return model.KafkaConfluentCost{}, fmt.Errorf("no costs found for date %s", date)
	}
	kafkaCosts, ok := costsForDay.kafka[clusterId]
	if !ok {
		return model.KafkaConfluentCost{}, fmt.Errorf("no costs found for cluster %s", clusterId)
	}

	costOfType, ok := kafkaCosts[costType]
	if !ok {
		return model.KafkaConfluentCost{}, fmt.Errorf("no costs found for cost type: %s", costType)
	}
	return costOfType, nil
}

func (c *ConfluentCostService) HasCostsForDate(date util.YearMonthDayDate) bool {
	_, ok := c.cachedCosts[date]
	return ok
}

func (c *ConfluentCostService) FetchAndCacheCosts(dayTime util.YearMonthDayDate) {
	costs, err := c.getCostsForDate(dayTime)
	if err != nil {
		log.Err(err).Msgf("failed to get costs for date %s", dayTime)
		return
	}
	c.CacheCosts(dayTime, costs)
}

func (c *ConfluentCostService) getCostsForDate(date util.YearMonthDayDate) (model.ConfluentCostResponse, error) {

	toTime := date.ToTimeUTC()
	fromTime := toTime.Add(-24 * time.Hour)
	costs, err := c.confluentCloudClient.GetCosts(fromTime, toTime)
	if err != nil {
		return model.ConfluentCostResponse{}, err
	}

	return *costs, nil
}
