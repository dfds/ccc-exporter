package model

import "fmt"

const ConfluentCostKafkaStorageReplicationFactor = 3

type CostType string

const (
	CostTypeConnectCapable    CostType = "CONNECT_CAPACITY"
	CostTypeConnectNumTasks   CostType = "CONNECT_NUM_TASKS"
	CostTypeConnectThroughPut CostType = "CONNECT_THROUGHPUT"
	CostTypeKafkaBase         CostType = "KAFKA_BASE"
	CostTypeKafkaStorage      CostType = "KAFKA_STORAGE"
	CostTypeKafkaNetworkRead  CostType = "KAFKA_NETWORK_READ"
	CostTypeKafkaNetworkWrite CostType = "KAFKA_NETWORK_WRITE"
	CostTypeKafkaNumCkus      CostType = "KAFKA_NUM_CKUS"
	CostTypeKafkaPartition    CostType = "KAFKA_PARTITION"
	CostTypeSupport           CostType = "SUPPORT"
)

var CostTypes = []CostType{
	CostTypeConnectCapable,
	CostTypeConnectNumTasks,
	CostTypeConnectThroughPut,
	CostTypeKafkaBase,
	CostTypeKafkaStorage,
	CostTypeKafkaNetworkRead,
	CostTypeKafkaNetworkWrite,
	CostTypeKafkaNumCkus,
	CostTypeKafkaPartition,
	CostTypeSupport,
}

func TryParseCostType(s string) (CostType, error) {
	for _, costType := range CostTypes {
		if s == string(costType) {
			return costType, nil
		}
	}
	return "", fmt.Errorf("invalid cost type: %s", s)
}

type ProductType string

const (
	ProductTypeConnect ProductType = "CONNECT"
	ProductTypeKafka   ProductType = "KAFKA"
	ProductTypeSupport ProductType = "SUPPORT_CLOUD_BUSINESS"
)

var ProductTypes = []ProductType{ProductTypeConnect, ProductTypeKafka, ProductTypeSupport}

func TryParseProductType(s string) (ProductType, error) {
	for _, productType := range ProductTypes {
		if s == string(productType) {
			return productType, nil
		}
	}
	return "", fmt.Errorf("invalid product type: %s", s)

}

type CostUnit string

const (
	GB            = "GB"
	GBHour        = "GB-hour"
	Hour          = "Hour"
	CKUHour       = "CKU-hour"
	TaskHour      = "Task-hour"
	PartitionHour = "Partition-hour"
)

var CostUnits = []CostUnit{GB, GBHour, Hour, CKUHour, TaskHour, PartitionHour}

func TryParseCostUnit(s string) (CostUnit, error) {
	for _, costUnit := range CostUnits {
		if s == string(costUnit) {
			return costUnit, nil
		}
	}
	return "", fmt.Errorf("invalid cost unit: %s", s)
}

type ConfluentCost struct {
	CostType    CostType
	ProductType ProductType
	ClusterId   ClusterId
	CostPerUnit float64
	CostUnit    CostUnit
	TotalCost   float64
}

type ConfluentCostResponse struct {
	ApiVersion string `json:"api_version"`
	Data       []struct {
		Amount            float64 `json:"amount"`
		EndDate           string  `json:"end_date"`
		Granularity       string  `json:"granularity"`
		LineType          string  `json:"line_type"`
		OriginalAmount    float64 `json:"original_amount"`
		Product           string  `json:"product"`
		StartDate         string  `json:"start_date"`
		DiscountAmount    float64 `json:"discount_amount,omitempty"`
		NetworkAccessType string  `json:"network_access_type,omitempty"`
		Price             float64 `json:"price,omitempty"`
		Quantity          float64 `json:"quantity,omitempty"`
		Resource          struct {
			DisplayName string `json:"display_name"`
			Environment struct {
				Id string `json:"id"`
			} `json:"environment"`
			Id string `json:"id"`
		} `json:"resource,omitempty"`
		Unit string `json:"unit,omitempty"`
	} `json:"data"`
	Kind     string `json:"kind"`
	Metadata struct {
		Next string `json:"next"`
	} `json:"metadata"`
}
