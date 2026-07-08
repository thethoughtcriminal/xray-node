package panel

import "encoding/json"

type APIResponse struct {
	Success bool            `json:"success"`
	Msg     string          `json:"msg"`
	Obj     json.RawMessage `json:"obj"`
}

// Inbound mirrors the 3x-ui inbound list item.
type Inbound struct {
	ID             int    `json:"id"`
	Up             int64  `json:"up"`
	Down           int64  `json:"down"`
	Total          int64  `json:"total"`
	Remark         string `json:"remark"`
	Enable         bool   `json:"enable"`
	ExpiryTime     int64  `json:"expiryTime"`
	Listen         string `json:"listen"`
	Port           int    `json:"port"`
	Protocol       string `json:"protocol"`
	Settings       JSONField `json:"settings"`
	StreamSettings JSONField `json:"streamSettings"`
	Tag            string    `json:"tag"`
	Sniffing       JSONField `json:"sniffing"`
	Allocate       JSONField `json:"allocate"`
}

type ClientTraffic struct {
	ID         int    `json:"id"`
	InboundID  int    `json:"inboundId"`
	Enable     bool   `json:"enable"`
	Email      string `json:"email"`
	Up         int64  `json:"up"`
	Down       int64  `json:"down"`
	ExpiryTime int64  `json:"expiryTime"`
	Total      int64  `json:"total"`
	Reset      int    `json:"reset"`
}

type ClientSettings struct {
	Clients []Client `json:"clients"`
}

type Client struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Enable     bool   `json:"enable"`
	ExpiryTime int64  `json:"expiryTime"`
	TotalGB    int64  `json:"totalGB"`
	LimitIP    int    `json:"limitIp"`
	SubID      string `json:"subId"`
	Flow       string `json:"flow,omitempty"`
	Auth       string `json:"auth,omitempty"`
	Password   string `json:"password,omitempty"`
	Comment    string `json:"comment,omitempty"`
}
