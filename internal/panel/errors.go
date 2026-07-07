package panel

import (
	"errors"
	"fmt"
	"net/http"
)

// HTTPError is returned when the panel responds with a non-2xx status.
type HTTPError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("panel %s %s: HTTP %d (check panel.token and panel.url base path)", e.Method, e.Path, e.Status)
	}
	return fmt.Sprintf("panel %s %s: %s", e.Method, e.Path, e.Body)
}

func isHTTP404(err error) bool {
	var he *HTTPError
	return errors.As(err, &he) && he.Status == http.StatusNotFound
}
