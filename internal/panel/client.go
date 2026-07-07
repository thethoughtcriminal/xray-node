package panel

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type PanelClient struct {
	baseURL     string
	token       string
	username    string
	password    string
	insecureTLS bool
	http        *http.Client
	loggedIn    bool
}

func New(baseURL, token, username, password string, insecureTLS bool) *PanelClient {
	jar, _ := cookiejar.New(nil)
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureTLS}, //nolint:gosec
	}
	return &PanelClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		token:       token,
		username:    username,
		password:    password,
		insecureTLS: insecureTLS,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Jar:       jar,
			Transport: transport,
		},
	}
}

func (c *PanelClient) ListInbounds() ([]Inbound, error) {
	var out []Inbound
	if err := c.getJSON("/panel/api/inbounds/list", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *PanelClient) FindInboundByRemark(remark string) (*Inbound, error) {
	items, err := c.ListInbounds()
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Remark == remark {
			return &items[i], nil
		}
	}
	return nil, nil
}

func (c *PanelClient) AddInbound(payload map[string]any) (*Inbound, error) {
	var created Inbound
	if err := c.postJSON("/panel/api/inbounds/add", payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *PanelClient) UpdateInbound(id int, payload map[string]any) (*Inbound, error) {
	var updated Inbound
	path := fmt.Sprintf("/panel/api/inbounds/update/%d", id)
	if err := c.postJSON(path, payload, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *PanelClient) AddClient(inboundID int, client Client) error {
	body := map[string]any{
		"client":     clientPayload(client),
		"inboundIds": []int{inboundID},
	}
	if err := c.postJSON("/panel/api/clients/add", body, nil); err == nil {
		return nil
	} else if !isHTTP404(err) {
		return err
	}
	return c.addClientLegacy(inboundID, client)
}

func (c *PanelClient) addClientLegacy(inboundID int, client Client) error {
	settings, err := json.Marshal(ClientSettings{Clients: []Client{client}})
	if err != nil {
		return err
	}
	body := map[string]any{
		"id":       inboundID,
		"settings": string(settings),
	}
	return c.postJSON("/panel/api/inbounds/addClient", body, nil)
}

func (c *PanelClient) UpdateClient(inboundID int, client Client) error {
	body := clientPayload(client)
	body["email"] = client.Email
	path := fmt.Sprintf("/panel/api/clients/update/%s", url.PathEscape(client.Email))
	if err := c.postJSON(path, body, nil); err == nil {
		return nil
	} else if !isHTTP404(err) {
		return err
	}
	return c.updateClientLegacy(inboundID, client)
}

func (c *PanelClient) updateClientLegacy(inboundID int, client Client) error {
	settings, err := json.Marshal(ClientSettings{Clients: []Client{client}})
	if err != nil {
		return err
	}
	body := map[string]any{
		"id":       inboundID,
		"settings": string(settings),
	}
	clientID := clientClientID(client)
	path := fmt.Sprintf("/panel/api/inbounds/updateClient/%s", url.PathEscape(clientID))
	return c.postJSON(path, body, nil)
}

func (c *PanelClient) SetClientEnabled(inboundID int, client Client, enabled bool) error {
	client.Enable = enabled
	return c.UpdateClient(inboundID, client)
}

func (c *PanelClient) GetClientTraffic(email string) (*ClientTraffic, error) {
	var traffic ClientTraffic
	path := fmt.Sprintf("/panel/api/clients/traffic/%s", url.PathEscape(email))
	if err := c.getJSON(path, &traffic); err == nil {
		return &traffic, nil
	} else if !isHTTP404(err) {
		return nil, err
	}
	legacyPath := fmt.Sprintf("/panel/api/inbounds/getClientTraffics/%s", url.PathEscape(email))
	if err := c.getJSON(legacyPath, &traffic); err != nil {
		return nil, err
	}
	return &traffic, nil
}

func (c *PanelClient) GetInboundClients(inbound Inbound) ([]Client, error) {
	var settings ClientSettings
	if err := json.Unmarshal([]byte(inbound.Settings.String()), &settings); err != nil {
		return nil, fmt.Errorf("parse inbound settings: %w", err)
	}
	return settings.Clients, nil
}

func (c *PanelClient) FindClient(inbound Inbound, email string) (*Client, error) {
	clients, err := c.GetInboundClients(inbound)
	if err != nil {
		return nil, err
	}
	for i := range clients {
		if clients[i].Email == email {
			return &clients[i], nil
		}
	}
	return nil, nil
}

func clientClientID(c Client) string {
	if c.ID != "" {
		return c.ID
	}
	if c.Password != "" {
		return c.Password
	}
	return c.Email
}

func clientPayload(client Client) map[string]any {
	payload := map[string]any{
		"email":      client.Email,
		"enable":     client.Enable,
		"expiryTime": client.ExpiryTime,
		"totalGB":    client.TotalGB,
		"limitIp":    client.LimitIP,
	}
	if client.ID != "" {
		payload["uuid"] = client.ID
	}
	if client.SubID != "" {
		payload["subId"] = client.SubID
	}
	if client.Flow != "" {
		payload["flow"] = client.Flow
	}
	if client.Auth != "" {
		payload["auth"] = client.Auth
	}
	if client.Password != "" {
		payload["password"] = client.Password
	}
	return payload
}

func (c *PanelClient) getJSON(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *PanelClient) postJSON(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *PanelClient) do(req *http.Request, out any) error {
	if err := c.auth(req); err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return &HTTPError{
			Method: req.Method,
			Path:   req.URL.Path,
			Status: resp.StatusCode,
			Body:   strings.TrimSpace(string(raw)),
		}
	}

	var envelope APIResponse
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("decode panel response: %w", err)
	}
	if !envelope.Success {
		if envelope.Msg != "" {
			return fmt.Errorf("panel error: %s", envelope.Msg)
		}
		return fmt.Errorf("panel request failed")
	}
	if out == nil {
		return nil
	}
	return envelope.UnmarshalObj(out)
}

func (c *PanelClient) auth(req *http.Request) error {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		return nil
	}
	if c.loggedIn {
		return nil
	}
	if c.username == "" || c.password == "" {
		return fmt.Errorf("panel token or username/password required")
	}
	loginBody, _ := json.Marshal(map[string]string{
		"username": c.username,
		"password": c.password,
	})
	loginReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/login", bytes.NewReader(loginBody))
	if err != nil {
		return err
	}
	loginReq.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(loginReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("panel login failed: %s", strings.TrimSpace(string(raw)))
	}
	c.loggedIn = true
	return nil
}
