package bridge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client talks to the Java LoginBridge on 127.0.0.1.
type Client struct {
	Base  string
	Token string
	HTTP  *http.Client
}

func NewClient(base string) *Client {
	return &Client{
		Base: base,
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

type HealthResponse struct {
	OK      bool   `json:"ok"`
	Service string `json:"service,omitempty"`
	Bind    string `json:"bind,omitempty"`
	Version string `json:"version,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	OK        bool   `json:"ok"`
	Token     string `json:"token"`
	AccountID int    `json:"accountId"`
	Gender    int    `json:"gender"`
	GM        int    `json:"gm"`
	Error     string `json:"error,omitempty"`
}

type ChannelInfo struct {
	ID   int    `json:"id"`
	Load int    `json:"load"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type WorldInfo struct {
	ID       int           `json:"id"`
	Name     string        `json:"name,omitempty"`
	Channels []ChannelInfo `json:"channels"`
}

type WorldsResponse struct {
	OK     bool        `json:"ok"`
	Worlds []WorldInfo `json:"worlds"`
	Error  string      `json:"error,omitempty"`
}

type CharacterInfo struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Level  int    `json:"level,omitempty"`
	Job    int    `json:"job,omitempty"`
	World  int    `json:"world,omitempty"`
}

type CharactersResponse struct {
	OK         bool            `json:"ok"`
	Characters []CharacterInfo `json:"characters"`
	Error      string          `json:"error,omitempty"`
}

type SelectRequest struct {
	CharacterID int `json:"characterId"`
	Channel     int `json:"channel"`
}

type SelectResponse struct {
	OK          bool   `json:"ok"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	AuthIP      string `json:"authIp,omitempty"`
	CharacterID int    `json:"characterId,omitempty"`
	Channel     int    `json:"channel,omitempty"`
	Error       string `json:"error,omitempty"`
}

func (c *Client) Health() (*HealthResponse, error) {
	var out HealthResponse
	if err := c.doJSON(http.MethodGet, "/health", nil, false, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Login(username, password string) (*LoginResponse, error) {
	var out LoginResponse
	if err := c.doJSON(http.MethodPost, "/api/login", LoginRequest{Username: username, Password: password}, false, &out); err != nil {
		return nil, err
	}
	if out.OK && out.Token != "" {
		c.Token = out.Token
	}
	return &out, nil
}

func (c *Client) Worlds() (*WorldsResponse, error) {
	var out WorldsResponse
	if err := c.doJSON(http.MethodGet, "/api/worlds", nil, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Characters(world int) (*CharactersResponse, error) {
	path := "/api/characters?world=" + url.QueryEscape(strconv.Itoa(world))
	var out CharactersResponse
	if err := c.doJSON(http.MethodGet, path, nil, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Select(characterID, channel int) (*SelectResponse, error) {
	var out SelectResponse
	if err := c.doJSON(http.MethodPost, "/api/select", SelectRequest{CharacterID: characterID, Channel: channel}, true, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) doJSON(method, path string, body any, auth bool, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.Base+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		if c.Token == "" {
			return fmt.Errorf("missing bridge token; login first")
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, truncate(string(data), 200))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
