package middleware

import (
	"net/http"
	"sync/atomic"
)

type MetricsMiddleware struct {
	Hits *atomic.Int32
}

func NewMetricsMiddleware() *MetricsMiddleware {
	return &MetricsMiddleware{
		Hits: &atomic.Int32{},
	}
}

func (m *MetricsMiddleware) IncHits(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.Hits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (m *MetricsMiddleware) GetHits() int32 {
	return m.Hits.Load()
}
