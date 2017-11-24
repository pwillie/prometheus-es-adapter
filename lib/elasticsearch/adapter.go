package elasticsearch

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	elastic "gopkg.in/olivere/elastic.v5"
)

const sampleType = "sample"
const indexMapping = `{
	"mappings":{
		"sample": {
			"dynamic_templates": [
				{
					"strings": {
						"match_mapping_type": "string",
						"path_match": "label.*",
						"mapping": {
							"type": "keyword"
						}
					}
				}
			]
		}
	}
}`

type Sample struct {
	Labels    model.Metric `json:"label"`
	Value     float64      `json:"value"`
	Timestamp int64        `json:"timestamp"`
}

// An AdapterOptionFunc is a function that configures a Client.
// It is used in NewClient.
type AdapterOptionFunc func(*Adapter) error

type Adapter struct {
	c             *elastic.Client
	b             *elastic.BulkProcessor
	batchCount    int
	batchSize     int
	batchInterval int
	esIndex       string
	esUrl         string
	workers       int
	stats         bool
}

// NewAdapter creates and returns a new elasticsearch adapter
func NewAdapter(options ...AdapterOptionFunc) (*Adapter, error) {
	a := &Adapter{
		batchCount:    1000,
		batchSize:     4096,
		batchInterval: 10,
		stats:         true,
	}
	// Run the options
	for _, option := range options {
		if err := option(a); err != nil {
			return nil, err
		}
	}

	client, err := elastic.NewClient(
		elastic.SetURL(a.esUrl),
		elastic.SetBasicAuth("elastic", "changeme"),
		elastic.SetSniff(false),
	)
	if err != nil {
		log.Println("Failed to create elastic client")
		log.Fatal(err)
	}
	defer client.Stop()

	a.c = client

	ctx := context.Background()

	a.ensureIndex(ctx, &a.esIndex)

	b, err := client.BulkProcessor().
		Workers(a.workers).                                          // # of workers
		BulkActions(a.batchCount).                                   // # of queued requests before committed
		BulkSize(a.batchSize).                                       // # of bytes in requests before committed
		FlushInterval(time.Duration(a.batchInterval) * time.Second). // autocommit every # seconds
		Stats(a.stats).                                              // gather statistics
		// Before(b.before).                // call "before" before every commit
		After(a.after). // call "after" after every commit
		Do(ctx)

	a.b = b
	if a.stats {
		prometheus.MustRegister(a)
	}
	return a, nil
}

func SetBatchCount(samples int) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.batchCount = samples
		return nil
	}
}

func SetBatchSize(bytes int) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.batchSize = bytes
		return nil
	}
}

func SetBatchInterval(seconds int) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.batchInterval = seconds
		return nil
	}
}

func SetEsIndex(index string) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.esIndex = index
		return nil
	}
}

func SetEsUrl(url string) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.esUrl = url
		return nil
	}
}

func SetStats(enabled bool) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.stats = enabled
		return nil
	}
}

func SetWorkers(workers int) AdapterOptionFunc {
	return func(a *Adapter) error {
		a.workers = workers
		return nil
	}
}

// after is invoked by bulk processor after every commit.
// The err variable indicates success or failure.
func (a *Adapter) after(id int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
	for _, i := range response.Items {
		if i["index"].Status != 201 {
			spew.Dump(i["index"])
		}
	}
}

func (a *Adapter) Close() error {
	return a.b.Close()
}

func (a *Adapter) Write(req []*prompb.TimeSeries) error {
	for _, ts := range req {
		metric := make(model.Metric, len(ts.Labels))
		for _, l := range ts.Labels {
			metric[model.LabelName(l.Name)] = model.LabelValue(l.Value)
		}
		for _, s := range ts.Samples {
			v := float64(s.Value)
			if math.IsNaN(v) || math.IsInf(v, 0) {
				// TODO: debug maybe? log.Println("invalid value, skipping sample", "value", v, "sample", s)
				continue
			}
			sample := Sample{
				metric,
				v,
				s.Timestamp,
			}
			r := elastic.
				NewBulkIndexRequest().
				Index(a.esIndex).
				Type(sampleType).
				Doc(sample)
			a.b.Add(r)
		}
	}
	return nil
}

func (a *Adapter) Read(req []*prompb.Query) ([]*prompb.QueryResult, error) {
	spew.Dump(req)
	results := make([]*prompb.QueryResult, 0, len(req))
	for _, q := range req {
		command := a.buildCommand(q)

		resp, err := command.Do(context.Background())
		if err != nil {
			return nil, err
		}
		log.Printf("Query returned %d results", resp.Hits.TotalHits)
		ts, err := createTimeseries(resp.Hits)
		if err != nil {
			return nil, err
		}
		results = append(results, &prompb.QueryResult{Timeseries: ts})
	}
	return results, nil
}

func (a *Adapter) buildCommand(q *prompb.Query) *elastic.SearchService {

	query := elastic.NewBoolQuery()
	for _, m := range q.Matchers {
		switch m.Type {
		case prompb.LabelMatcher_EQ:
			query = query.Filter(elastic.NewTermQuery("label."+m.Name, m.Value))
		// case prompb.LabelMatcher_NEQ:
		// case prompb.LabelMatcher_RE:
		// case prompb.LabelMatcher_NRE:
		default:
			log.Panicf("unknown match type %v", m.Type)
		}
	}

	query = query.Filter(elastic.NewRangeQuery("timestamp").Gte(q.StartTimestampMs).Lte(q.EndTimestampMs))

	// ss, _ := elastic.NewSearchSource().Query(query).Source()
	// data, _ := json.Marshal(ss)
	// fmt.Printf("%s", string(data))

	service := a.c.Search().Index(a.esIndex).Type(sampleType).Query(query).Size(1000).Sort("timestamp", true)
	return service
}

func createTimeseries(results *elastic.SearchHits) ([]*prompb.TimeSeries, error) {
	tsMap := make(map[string]*prompb.TimeSeries)
	for _, r := range results.Hits {
		var s Sample
		if err := json.Unmarshal([]byte(*r.Source), &s); err != nil {
			log.Fatal(err)
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
		ts.Samples = append(ts.Samples, &prompb.Sample{
			Value:     s.Value,
			Timestamp: s.Timestamp,
		})
	}
	ret := make([]*prompb.TimeSeries, 0, len(tsMap))

	for _, s := range tsMap {
		ret = append(ret, s)
	}
	spew.Dump(ret)
	return ret, nil
}

// ensureIndex creates the index in Elasticsearch.
func (a *Adapter) ensureIndex(ctx context.Context, index *string) error {
	if *index == "" {
		return errors.New("no index name")
	}
	exists, err := a.c.IndexExists(*index).Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		_, err = a.c.CreateIndex(*index).BodyString(indexMapping).Do(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
