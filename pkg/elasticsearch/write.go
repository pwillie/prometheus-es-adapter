package elasticsearch

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
	elastic "gopkg.in/olivere/elastic.v6"
)

type prometheusSample struct {
	Labels    model.Metric `json:"label"`
	Value     float64      `json:"value"`
	Timestamp int64        `json:"timestamp"`
}

// WriteService will proxy Prometheus write requests to Elasticsearch
type WriteService struct {
	config    *WriteConfig
	logger    *zap.Logger
	processor *elastic.BulkProcessor
}

// WriteConfig is used to configure WriteService
type WriteConfig struct {
	Alias   string
	MaxAge  int
	MaxDocs int
	MaxSize int
	Workers int
	Stats   bool
}

// NewWriteService creates and returns a new elasticsearch WriteService
func NewWriteService(ctx context.Context, logger *zap.Logger, client *elastic.Client, config *WriteConfig) (*WriteService, error) {
	svc := &WriteService{
		config: config,
		logger: logger,
	}
	b, err := client.BulkProcessor().
		Workers(config.Workers).                                   // # of workers
		BulkActions(config.MaxDocs).                               // # of queued requests before committed
		BulkSize(config.MaxSize).                                  // # of bytes in requests before committed
		FlushInterval(time.Duration(config.MaxAge) * time.Second). // autocommit every # seconds
		Stats(config.Stats).                                       // gather statistics
		After(svc.after).                                          // call "after" after every commit
		Do(ctx)
	if err != nil {
		return nil, err
	}
	svc.processor = b
	if config.Stats {
		prometheus.MustRegister(svc)
	}
	return svc, nil
}

// Close will close the underlying elasticsearch BulkProcessor
func (svc *WriteService) Close() error {
	return svc.processor.Close()
}

// Write will enqueue Prometheus sample data to be batch written to Elasticsearch
func (svc *WriteService) Write(req []*prompb.TimeSeries) {
	for _, ts := range req {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		for _, s := range ts.Samples {
			v := float64(s.Value)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				svc.logger.Debug(fmt.Sprintf("invalid value %+v, skipping sample %+v", v, s))
				continue
			}
			sample := prometheusSample{
				metric,
				v,
				s.Timestamp,
			}
			r := elastic.
				NewBulkIndexRequest().
				Index(svc.config.Alias).
				Type(sampleType).
				Doc(sample)
			svc.processor.Add(r)
		}
	}
}

// after is invoked by bulk processor after every commit.
// The err variable indicates success or failure.
func (svc *WriteService) after(id int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	for _, i := range response.Items {
		if i["index"].Status != 201 {
			svc.logger.Error(fmt.Sprintf("%+v", i["index"].Error))
		}
	}
}
