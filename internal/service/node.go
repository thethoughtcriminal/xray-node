package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/thethoughtcriminal/xray-node/internal/config"
	"github.com/thethoughtcriminal/xray-node/internal/inbound"
	"github.com/thethoughtcriminal/xray-node/internal/panel"
)

type Node struct {
	panel *panel.PanelClient
}

func New(cfg *config.Config) *Node {
	return &Node{
		panel: panel.New(
			cfg.Panel.URL,
			cfg.Panel.Token,
			cfg.Panel.Username,
			cfg.Panel.Password,
			cfg.Panel.InsecureTLS,
		),
	}
}

func (n *Node) ApplyInbound(spec *inbound.Spec) (*panel.Inbound, error) {
	settings, err := spec.SettingsJSON()
	if err != nil {
		return nil, err
	}
	stream, err := spec.StreamSettingsJSON()
	if err != nil {
		return nil, err
	}
	sniffing, err := spec.SniffingJSON()
	if err != nil {
		return nil, err
	}
	allocate, err := spec.AllocateJSON()
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"remark":         spec.Remark,
		"protocol":       spec.Protocol,
		"listen":         spec.Listen,
		"port":           spec.Port,
		"enable":         *spec.Enable,
		"settings":       settings,
		"streamSettings": stream,
	}
	if spec.Tag != "" {
		payload["tag"] = spec.Tag
	}
	if sniffing != "" {
		payload["sniffing"] = sniffing
	}
	if allocate != "" {
		payload["allocate"] = allocate
	}

	existing, err := n.panel.FindInboundByRemark(spec.Remark)
	if err != nil {
		return nil, err
	}
	if existing != nil && spec.IsRealityVLESS() && !spec.HasRealityKeys() {
		if err := spec.MergeRealityKeysFromStream(existing.StreamSettings.String()); err != nil {
			return nil, err
		}
	}
	if spec.IsRealityVLESS() && !spec.HasRealityKeys() {
		if err := spec.EnsureRealityKeys(); err != nil {
			return nil, err
		}
		stream, err = spec.StreamSettingsJSON()
		if err != nil {
			return nil, err
		}
		payload["streamSettings"] = stream
	}
	if existing == nil {
		return n.panel.AddInbound(payload)
	}

	// Preserve clients on update unless explicitly provided in config.
	if clients, ok := spec.Settings["clients"].([]any); !ok || len(clients) == 0 {
		var current panel.ClientSettings
		if err := json.Unmarshal([]byte(existing.Settings.String()), &current); err == nil && len(current.Clients) > 0 {
			spec.Settings["clients"] = current.Clients
			settings, err = spec.SettingsJSON()
			if err != nil {
				return nil, err
			}
			payload["settings"] = settings
		}
	}

	payload["id"] = existing.ID
	payload["up"] = existing.Up
	payload["down"] = existing.Down
	payload["total"] = existing.Total
	payload["expiryTime"] = existing.ExpiryTime
	return n.panel.UpdateInbound(existing.ID, payload)
}

type AddClientInput struct {
	InboundRemark string
	Email         string
	UUID          string
	SubID         string
	Flow          string
	Auth          string
	TotalGB       int64
	ExpiryDays    int
	LimitIP       int
	Enable        bool
}

func (n *Node) AddClient(in AddClientInput) (*panel.Client, error) {
	if in.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	inboundRec, err := n.resolveInbound(in.InboundRemark)
	if err != nil {
		return nil, err
	}
	if existing, err := n.panel.FindClient(*inboundRec, in.Email); err != nil {
		return nil, err
	} else if existing != nil {
		return nil, fmt.Errorf("client %q already exists on inbound %q", in.Email, inboundRec.Remark)
	}

	client := panel.Client{
		Email:      in.Email,
		Enable:     in.Enable,
		ExpiryTime: expiryFromDays(in.ExpiryDays),
		TotalGB:    in.TotalGB,
		LimitIP:    in.LimitIP,
		SubID:      in.SubID,
		Flow:       in.Flow,
		Auth:       in.Auth,
	}
	if in.UUID != "" {
		client.ID = in.UUID
	} else if inboundRec.Protocol == "vless" || inboundRec.Protocol == "vmess" {
		client.ID = uuid.NewString()
	}
	if client.SubID == "" {
		client.SubID = uuid.NewString()[:16]
	}
	if inboundRec.Protocol == "vless" && client.Flow == "" {
		client.Flow = "xtls-rprx-vision"
	}
	if inboundRec.Protocol == "hysteria" && client.Auth == "" {
		client.Auth = client.ID
		if client.Auth == "" {
			client.Auth = uuid.NewString()
		}
	}

	if err := n.panel.AddClient(inboundRec.ID, client); err != nil {
		return nil, err
	}
	return &client, nil
}

func (n *Node) SetClientEnabled(inboundRemark, email string, enabled bool) error {
	inboundRec, client, err := n.findClient(inboundRemark, email)
	if err != nil {
		return err
	}
	return n.panel.SetClientEnabled(inboundRec.ID, *client, enabled)
}

func (n *Node) ClientStats(inboundRemark, email string) (*panel.ClientTraffic, error) {
	if _, _, err := n.findClient(inboundRemark, email); err != nil {
		return nil, err
	}
	return n.panel.GetClientTraffic(email)
}

func (n *Node) ListInbounds() ([]panel.Inbound, error) {
	return n.panel.ListInbounds()
}

func (n *Node) findClient(inboundRemark, email string) (*panel.Inbound, *panel.Client, error) {
	inboundRec, err := n.resolveInbound(inboundRemark)
	if err != nil {
		return nil, nil, err
	}
	client, err := n.panel.FindClient(*inboundRec, email)
	if err != nil {
		return nil, nil, err
	}
	if client == nil {
		return nil, nil, fmt.Errorf("client %q not found on inbound %q", email, inboundRec.Remark)
	}
	return inboundRec, client, nil
}

func (n *Node) resolveInbound(remark string) (*panel.Inbound, error) {
	if remark == "" {
		return nil, fmt.Errorf("inbound remark is required")
	}
	inboundRec, err := n.panel.FindInboundByRemark(remark)
	if err != nil {
		return nil, err
	}
	if inboundRec == nil {
		return nil, fmt.Errorf("inbound %q not found", remark)
	}
	return inboundRec, nil
}

func expiryFromDays(days int) int64 {
	if days <= 0 {
		return 0
	}
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).UnixMilli()
}
