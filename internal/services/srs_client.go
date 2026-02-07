package services

import (
	"encoding/json"
	"net/http"
)

type SRSClient struct {
	baseURL string
}

func NewSRSClient() *SRSClient {
	return &SRSClient{
		baseURL: "http://srs:1985/api/v1",
	}
}

func (c *SRSClient) GetStreams() (map[string]interface{}, error) {
	resp, err := http.Get(c.baseURL + "/streams/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *SRSClient) GetClients() (map[string]interface{}, error) {
	resp, err := http.Get(c.baseURL + "/clients/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *SRSClient) GetRusages() (map[string]interface{}, error) {
	resp, err := http.Get(c.baseURL + "/rusages/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}