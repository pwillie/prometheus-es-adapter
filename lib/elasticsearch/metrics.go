package elasticsearch

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "es_adapter"
)

var (
	flushedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "flushed"),
		"Number of times the flush interval has been invoked",
		nil,
		nil,
	)
	committedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "committed"),
		"Number of times workers committed bulk requests",
		nil,
		nil,
	)
	indexedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "indexed"),
		"Number of requests indexed",
		nil,
		nil,
	)
	createdDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "created"),
		"Number of requests that ES reported as creates (201)",
		nil,
		nil,
	)
	updatedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "updated"),
		"Number of requests that ES reported as updates",
		nil,
		nil,
	)
	deletedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "deleted"),
		"Number of requests that ES reported as deletes",
		nil,
		nil,
	)
	succeededDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "succeeded"),
		"Number of requests that ES reported as successful",
		nil,
		nil,
	)
	failedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "failed"),
		"Number of requests that ES reported as failed",
		nil,
		nil,
	)
	// TODO: queued should be per worker...
	queuedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "queued"),
		"Number of requests queued",
		nil,
		nil,
	)

// TODO: need to instrument LastDuration per worker ??
// i.e. LastDuration time.Duration // duration of last commit
// ch <- prometheus.NewDesc(
// 	prometheus.BuildFQName(namespace, "", "duration"),
// 	"Duration of last commit",
// 	nil,
// 	nil,
// )
)

// Describe describes all the metrics exported by the memcached exporter. It
// implements prometheus.Collector.
func (e *Adapter) Describe(ch chan<- *prometheus.Desc) {
	ch <- flushedDesc
	ch <- committedDesc
	ch <- indexedDesc
	ch <- createdDesc
	ch <- updatedDesc
	ch <- deletedDesc
	ch <- succeededDesc
	ch <- failedDesc
	ch <- queuedDesc
}

// Collect fetches the statistics from the configured memcached server, and
// delivers them as Prometheus metrics. It implements prometheus.Collector.
func (a *Adapter) Collect(ch chan<- prometheus.Metric) {
	stats := a.b.Stats()

	var queued int64
	for _, w := range stats.Workers {
		queued += w.Queued
	}

	ch <- prometheus.MustNewConstMetric(flushedDesc, prometheus.CounterValue, float64(stats.Flushed))
	ch <- prometheus.MustNewConstMetric(committedDesc, prometheus.CounterValue, float64(stats.Committed))
	ch <- prometheus.MustNewConstMetric(indexedDesc, prometheus.CounterValue, float64(stats.Indexed))
	ch <- prometheus.MustNewConstMetric(createdDesc, prometheus.CounterValue, float64(stats.Created))
	ch <- prometheus.MustNewConstMetric(updatedDesc, prometheus.CounterValue, float64(stats.Updated))
	ch <- prometheus.MustNewConstMetric(deletedDesc, prometheus.CounterValue, float64(stats.Deleted))
	ch <- prometheus.MustNewConstMetric(succeededDesc, prometheus.CounterValue, float64(stats.Succeeded))
	ch <- prometheus.MustNewConstMetric(failedDesc, prometheus.CounterValue, float64(stats.Failed))
	ch <- prometheus.MustNewConstMetric(queuedDesc, prometheus.GaugeValue, float64(queued))
}
