package masterclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

type EnrollRequest struct {
	Token      string `json:"token"`
	Name       string `json:"name"`
	APIURL     string `json:"api_url"`
	APIKey     string `json:"api_key"`
	PublicHost string `json:"public_host"`
	IP         string `json:"ip"`
}

type EnrollResponse struct {
	ID         string `json:"ID"`
	Name       string `json:"Name"`
	APIURL     string `json:"APIURL"`
	PublicHost string `json:"PublicHost"`
	Status     string `json:"Status"`
}

func (c *Client) Enroll(req EnrollRequest) (*EnrollResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/nodes/enroll", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		var envelope struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &envelope)
		if envelope.Error != "" {
			return nil, fmt.Errorf("master enroll: %s", envelope.Error)
		}
		return nil, fmt.Errorf("master enroll: HTTP %d", resp.StatusCode)
	}
	var out EnrollResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
