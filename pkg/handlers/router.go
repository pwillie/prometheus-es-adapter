package handlers

import (
	"net/http"

	"gopkg.in/olivere/elastic.v6"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/pwillie/prometheus-es-adapter/pkg/elasticsearch"
)

// NewRouter returns a configured http router
func NewRouter(w *elasticsearch.WriteService, r *elasticsearch.ReadService) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/read", readHandler(r))
	mux.HandleFunc("/write", writeHandler(w))
	return mux
}

// NewAdminRouter returns a configured http router for prom metrics and health checks
func NewAdminRouter(client *elastic.Client) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", prometheus.Handler())
	// creates /live and /ready endpoints
	mux.Handle("/", healthzHandler(client))
	return mux
}
