package httpapi

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

type requestMetricKey struct {
	method string
	route  string
	status int
}

type requestMetrics struct {
	mu       sync.Mutex
	requests map[requestMetricKey]uint64
}

func newRequestMetrics() *requestMetrics {
	return &requestMetrics{requests: make(map[requestMetricKey]uint64)}
}

func (metrics *requestMetrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		method := r.Method
		switch method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead:
		default:
			method = "OTHER"
		}
		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}
		metrics.mu.Lock()
		metrics.requests[requestMetricKey{method: method, route: route, status: status}]++
		metrics.mu.Unlock()
	})
}

func (metrics *requestMetrics) handler(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" {
			const prefix = "Bearer "
			authorization := r.Header.Get("Authorization")
			if !strings.HasPrefix(authorization, prefix) ||
				subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(authorization, prefix)), []byte(token)) != 1 {
				writeError(w, r, http.StatusUnauthorized, "unauthenticated", "A valid metrics bearer token is required.")
				return
			}
		}
		metrics.mu.Lock()
		keys := make([]requestMetricKey, 0, len(metrics.requests))
		for key := range metrics.requests {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].route != keys[j].route {
				return keys[i].route < keys[j].route
			}
			if keys[i].method != keys[j].method {
				return keys[i].method < keys[j].method
			}
			return keys[i].status < keys[j].status
		})
		counts := make([]uint64, len(keys))
		for index, key := range keys {
			counts[index] = metrics.requests[key]
		}
		metrics.mu.Unlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = fmt.Fprintln(w, "# HELP http_requests_total Total HTTP requests handled by route and status.")
		_, _ = fmt.Fprintln(w, "# TYPE http_requests_total counter")
		for index, key := range keys {
			_, _ = fmt.Fprintf(w, "http_requests_total{method=%q,route=%q,status=%q} %s\n",
				key.method, key.route, strconv.Itoa(key.status), strconv.FormatUint(counts[index], 10))
		}
	}
}
