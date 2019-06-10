package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/prompb"
	"go.uber.org/zap"
	elastic "gopkg.in/olivere/elastic.v6"
)

// ReadService will proxy Prometheus queries to Elasticsearch
type ReadService struct {
	client *elastic.Client
	config *ReadConfig
	logger *zap.Logger
}

// ReadConfig configures the ReadService
type ReadConfig struct {
	Alias   string
	MaxDocs int
}

// NewReadService will create a new ReadService
func NewReadService(logger *zap.Logger, client *elastic.Client, config *ReadConfig) *ReadService {
	svc := &ReadService{
		client: client,
		config: config,
		logger: logger,
	}
	// TODO: add stats
	return svc
}

// Read will perform Elasticsearch query
func (svc *ReadService) Read(ctx context.Context, req []*prompb.Query) ([]*prompb.QueryResult, error) {
	results := make([]*prompb.QueryResult, 0, len(req))
	for _, q := range req {
		resp, err := svc.buildCommand(q).Do(ctx)
		if err != nil {
			return nil, err
		}
		svc.logger.Debug("Query returned results", zap.Int64("hits", resp.Hits.TotalHits))
		ts, err := svc.createTimeseries(resp.Hits)
		if err != nil {
			return nil, err
		}
		results = append(results, &prompb.QueryResult{Timeseries: ts})
	}
	return results, nil
}

func (svc *ReadService) buildCommand(q *prompb.Query) *elastic.SearchService {
	query := elastic.NewBoolQuery()
	for _, m := range q.Matchers {
		switch m.Type {
		case prompb.LabelMatcher_EQ:
			query = query.Filter(elastic.NewTermQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_NEQ:
			query = query.MustNot(elastic.NewTermQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_RE:
			query = query.Filter(elastic.NewRegexpQuery("label."+m.Name, m.Value))
		case prompb.LabelMatcher_NRE:
			query = query.MustNot(elastic.NewRegexpQuery("label."+m.Name, m.Value))
		default:
			svc.logger.Panic("unknown match", zap.String("type", m.Type.String()))
		}
	}

	query = query.Filter(elastic.NewRangeQuery("timestamp").Gte(q.StartTimestampMs).Lte(q.EndTimestampMs))

	return svc.client.Search().
		Index(svc.config.Alias+"-*").
		Type(sampleType).
		Query(query).
		Size(svc.config.MaxDocs).
		Sort("timestamp", true)
}

func (svc *ReadService) createTimeseries(results *elastic.SearchHits) ([]*prompb.TimeSeries, error) {
	tsMap := make(map[string]*prompb.TimeSeries)
	for _, r := range results.Hits {
		var s prometheusSample
		if err := json.Unmarshal([]byte(*r.Source), &s); err != nil {
			svc.logger.Fatal("Failed to unmarshal sample", zap.Error(err))
		}
		fingerprint := s.Labels.Fingerprint().String()

		ts, ok := tsMap[fingerprint]
		if !ok {
			labels := make([]*prompb.Label, 0, len(s.Labels))
			for k, v := range s.Labels {
				labels = append(labels, &prompb.Label{
					Name:  string(k),
					Value: string(v),
				})
			}
			ts = &prompb.TimeSeries{
				Labels: labels,
			}
			tsMap[fingerprint] = ts
		}
		ts.Samples = append(ts.Samples, prompb.Sample{
			Value:     s.Value,
			Timestamp: timestamp.FromTime(s.Timestamp),
		})
	}
	ret := make([]*prompb.TimeSeries, 0, len(tsMap))

	for _, s := range tsMap {
		ret = append(ret, s)
	}
	return ret, nil
}
