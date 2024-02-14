package client

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Client struct {
	endpoint string
	http     *http.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		endpoint: endpoint,
		http:     http.DefaultClient,
	}
}

func (c *Client) Query(query string, time float64) (*QueryResponse, error) {
	//req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/query_range", c.endpoint), nil)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/query", c.endpoint), nil)
	if err != nil {
		return nil, err
	}

	queryValues := req.URL.Query()
	queryValues.Set("query", query)
	queryValues.Set("time", fmt.Sprintf("%f", time))
	//queryValues.Set("start", "1704110693")
	//queryValues.Set("end", "1709250693")
	//queryValues.Set("step", "1d")
	req.URL.RawQuery = queryValues.Encode()
	fmt.Println(req.URL.String())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload *QueryResponse

	err = json.Unmarshal(data, &payload)

	return payload, err
}

func ResultToVector(data []interface{}) ([]Vector, error) {
	var casted []vectorMidParse
	for _, d := range data {
		var deserialised vectorMidParse
		serialised, err := json.Marshal(d)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(serialised, &deserialised)
		if err != nil {
			log.Fatal(err)
		}

		casted = append(casted, deserialised)
	}
	var payload []Vector

	for _, vec := range casted {
		newVec := Vector{
			Metric: VectorMetricLabel{
				Instance: vec.Metric.Instance,
				Job:      vec.Metric.Job,
				KafkaID:  vec.Metric.KafkaID,
				Topic:    vec.Metric.Topic,
			},
			Value: VectorValue{},
		}
		newVec.Value.Time = vec.Value[0].(float64)
		newVec.Value.Value = vec.Value[1].(string)
		payload = append(payload, newVec)
	}

	return payload, nil
}

type Vector struct {
	Metric VectorMetricLabel `json:"metric"`
	Value  VectorValue       `json:"value"`
}

type VectorMetricLabel struct {
	Instance string `json:"instance"`
	Job      string `json:"job"`
	KafkaID  string `json:"kafka_id"`
	Topic    string `json:"topic"`
}

type VectorValue struct {
	Time  float64
	Value string
}

type vectorMidParse struct {
	Metric struct {
		Instance string `json:"instance"`
		Job      string `json:"job"`
		KafkaID  string `json:"kafka_id"`
		Topic    string `json:"topic"`
	} `json:"metric"`
	Value []interface{} `json:"value"`
}

type QueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string        `json:"resultType"`
		Result     []interface{} `json:"result"`
	} `json:"data"`
}
