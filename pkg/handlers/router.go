package handlers

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/pwillie/prometheus-es-adapter/pkg/elasticsearch"
)

// NewRouter returns a configured http router
func NewRouter(w *elasticsearch.WriteService, r *elasticsearch.ReadService) *http.ServeMux {
	http.Handle("/metrics", prometheus.Handler())
	http.HandleFunc("/healthz", healthzHandler())
	http.HandleFunc("/read", readHandler(r))
	http.HandleFunc("/write", writeHandler(w))
	return http.DefaultServeMux
}

func healthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: monitor elastic.Client connection
		w.WriteHeader(http.StatusOK)
	}
}
