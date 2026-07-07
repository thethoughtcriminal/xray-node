package api

import (
	"errors"
	"net/http"

	"github.com/thethoughtcriminal/xray-node/internal/service"
)

func statusFromError(err error) int {
	switch {
	case errors.Is(err, service.ErrValidation):
		return http.StatusBadRequest
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		return http.StatusConflict
	default:
		return http.StatusBadGateway
	}
}

func writeServiceError(w http.ResponseWriter, err error) {
	writeError(w, statusFromError(err), err.Error())
}
