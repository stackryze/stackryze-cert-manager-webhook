package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Minimal Stackryze DNS API client (Bearer token).
type stackryzeClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newClient(baseURL, token string) *stackryzeClient {
	if baseURL == "" {
		baseURL = "https://api-dns.stackryze.com/api"
	}
	return &stackryzeClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

func strip(s string) string { return strings.TrimSuffix(s, ".") }

func (c *stackryzeClient) do(method, path string, body any) ([]byte, int, error) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return data, res.StatusCode, nil
}

type zone struct {
	ID   string `json:"_id"`
	Name string `json:"name"`
}

func (c *stackryzeClient) getZoneByName(name string) (*zone, error) {
	data, status, err := c.do("GET", "/zones", nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, fmt.Errorf("list zones failed: %s", string(data))
	}
	var resp struct {
		Zones []zone `json:"zones"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	name = strip(strings.ToLower(name))
	for i := range resp.Zones {
		if strip(strings.ToLower(resp.Zones[i].Name)) == name {
			return &resp.Zones[i], nil
		}
	}
	return nil, fmt.Errorf("zone %q not found on Stackryze", name)
}

func (c *stackryzeClient) addTXT(zoneID, label, content string) error {
	body := map[string]any{"type": "TXT", "name": label, "content": content, "ttl": 3600}
	data, status, err := c.do("POST", "/zones/"+zoneID+"/records", body)
	if err != nil {
		return err
	}
	// A repeated Present for the same challenge is not an error.
	if status == 400 && strings.Contains(strings.ToLower(string(data)), "already exists") {
		return nil
	}
	if status >= 400 {
		return fmt.Errorf("add TXT failed: %s", string(data))
	}
	return nil
}

func (c *stackryzeClient) deleteTXT(zoneID, label, content string) error {
	body := map[string]any{"type": "TXT", "name": label, "content": content}
	data, status, err := c.do("DELETE", "/zones/"+zoneID+"/records", body)
	if err != nil {
		return err
	}
	if status == 404 {
		return nil // already gone
	}
	if status >= 400 {
		return fmt.Errorf("delete TXT failed: %s", string(data))
	}
	return nil
}
