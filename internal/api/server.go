package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/thethoughtcriminal/xray-node/internal/config"
	"github.com/thethoughtcriminal/xray-node/internal/inbound"
	"github.com/thethoughtcriminal/xray-node/internal/service"
)

type Server struct {
	cfg  *config.Config
	node *service.Node
	http *http.Server
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:  cfg,
		node: service.New(cfg),
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, middleware.Logger)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.Get("/inbounds", s.listInbounds)
		r.Post("/inbounds/apply", s.applyInbound)
		r.Post("/clients", s.addClient)
		r.Post("/clients/{email}/enable", s.enableClient)
		r.Post("/clients/{email}/disable", s.disableClient)
		r.Get("/clients/{email}/stats", s.clientStats)
	})
	s.http = &http.Server{
		Addr:    cfg.API.Listen,
		Handler: r,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.API.Key == "" {
			writeError(w, http.StatusInternalServerError, "api key is not configured")
			return
		}
		if r.Header.Get("X-API-Key") != s.cfg.API.Key {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) listInbounds(w http.ResponseWriter, r *http.Request) {
	items, err := s.node.ListInbounds()
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) applyInbound(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var spec inbound.Spec
	if err := json.Unmarshal(body, &spec); err != nil {
		writeError(w, http.StatusBadRequest, "invalid inbound json: "+err.Error())
		return
	}
	if err := spec.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := s.node.ApplyInbound(&spec)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type addClientRequest struct {
	InboundRemark string `json:"inbound_remark"`
	Email         string `json:"email"`
	UUID          string `json:"uuid"`
	SubID         string `json:"sub_id"`
	Flow          string `json:"flow"`
	Auth          string `json:"auth"`
	Comment       string `json:"comment"`
	TotalGB       int64  `json:"total_gb"`
	ExpiryDays    int    `json:"expiry_days"`
	LimitIP       int    `json:"limit_ip"`
}

func (s *Server) addClient(w http.ResponseWriter, r *http.Request) {
	var req addClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	client, err := s.node.AddClient(service.AddClientInput{
		InboundRemark: req.InboundRemark,
		Email:         req.Email,
		UUID:          req.UUID,
		SubID:         req.SubID,
		Flow:          req.Flow,
		Auth:          req.Auth,
		Comment:       req.Comment,
		TotalGB:       req.TotalGB,
		ExpiryDays:    req.ExpiryDays,
		LimitIP:       req.LimitIP,
		Enable:        true,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, client)
}

func (s *Server) enableClient(w http.ResponseWriter, r *http.Request) {
	s.setClientEnabled(w, r, true)
}

func (s *Server) disableClient(w http.ResponseWriter, r *http.Request) {
	s.setClientEnabled(w, r, false)
}

func (s *Server) setClientEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	email := chi.URLParam(r, "email")
	remark := r.URL.Query().Get("inbound")
	if remark == "" {
		writeError(w, http.StatusBadRequest, "query param inbound is required")
		return
	}
	if err := s.node.SetClientEnabled(remark, email, enabled); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"email": email, "enabled": enabled})
}

func (s *Server) clientStats(w http.ResponseWriter, r *http.Request) {
	email := chi.URLParam(r, "email")
	remark := r.URL.Query().Get("inbound")
	if remark == "" {
		writeError(w, http.StatusBadRequest, "query param inbound is required")
		return
	}
	stats, err := s.node.ClientStats(remark, email)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
