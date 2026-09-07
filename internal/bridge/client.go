package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client talks to the Java LoginBridge on 127.0.0.1.
type Client struct {
	Base string
	HTTP *http.Client
}

func NewClient(base string) *Client {
	return &Client{
		Base: base,
		HTTP: &http.Client{Timeout: 5 * time.Second},
	}
}

type HealthResponse struct {
	OK      bool   `json:"ok"`
	Version string `json:"version,omitempty"`
}

func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.HTTP.Get(c.Base + "/health")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health status %d", resp.StatusCode)
	}
	var out HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Login / Worlds / Chars / SelectChar will be filled when LoginBridge API is finalized.
