package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/thethoughtcriminal/xray-node/internal/service"
)

func TestStatusFromError(t *testing.T) {
	tests := []struct {
		err    error
		status int
	}{
		{fmt.Errorf("%w: email required", service.ErrValidation), http.StatusBadRequest},
		{fmt.Errorf("%w: inbound missing", service.ErrNotFound), http.StatusNotFound},
		{fmt.Errorf("%w: client exists", service.ErrConflict), http.StatusConflict},
		{errors.New("panel timeout"), http.StatusBadGateway},
	}
	for _, tc := range tests {
		if got := statusFromError(tc.err); got != tc.status {
			t.Fatalf("statusFromError(%v) = %d, want %d", tc.err, got, tc.status)
		}
	}
}
