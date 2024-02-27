package client

import (
	"encoding/json"
	"fmt"
	"go.dfds.cloud/ccc-exporter/config"
	"go.dfds.cloud/ccc-exporter/internal/model"
	"io"
	"net/http"
	"time"
)

type ConfluentCloudClient struct {
	http   *http.Client
	config config.Confluent
}

func NewConfluentCloudClient(confluentConfig config.Confluent) *ConfluentCloudClient {
	return &ConfluentCloudClient{
		http:   http.DefaultClient,
		config: confluentConfig,
	}
}

func (c *ConfluentCloudClient) GetCosts(from time.Time, to time.Time) (*model.ConfluentCostResponse, error) {

	if from.After(to) {
		return nil, fmt.Errorf("from date is after to date")
	}

	req, err := http.NewRequest("GET", "https://api.confluent.cloud/billing/v1/costs", nil)
	if err != nil {
		return nil, err
	}

	queryValues := req.URL.Query()
	queryValues.Add("start_date", from.Format("2006-01-02"))
	queryValues.Add("end_date", from.Format("2006-01-02"))
	req.URL.RawQuery = queryValues.Encode()

	req.SetBasicAuth(c.config.ApiKeyId, c.config.ApiKeySecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got status code %d when getting confluent costs", resp.StatusCode)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload *model.ConfluentCostResponse
	err = json.Unmarshal(data, &payload)

	return payload, err
}
