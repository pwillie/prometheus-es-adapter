package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"go.uber.org/zap"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"github.com/pwillie/prometheus-es-adapter/lib/elasticsearch"
	"github.com/pwillie/prometheus-es-adapter/lib/logger"
)

// Main entry point.
func main() {
	var (
		url           = flag.String("es_url", "http://localhost:9200", "Elasticsearch URL.")
		user          = flag.String("es_user", "", "Elasticsearch User.")
		pass          = flag.String("es_password", "", "Elasticsearch User Password.")
		workers       = flag.Int("es_workers", 0, "Number of batch workers.")
		batchCount    = flag.Int("es_batch_count", 1000, "Max items for bulk Elasticsearch insert operation")
		batchSize     = flag.Int("es_batch_size", 4096, "Max size in bytes for bulk Elasticsearch insert operation")
		batchInterval = flag.Int("es_batch_interval", 10, "Max period in seconds between bulk Elasticsearch insert operations")
		indexMaxAge   = flag.String("es_index_max_age", "7d", "Max age of Elasticsearch index before rollover")
		indexMaxDocs  = flag.Int64("es_index_max_docs", 1000000, "Max number of docs in Elasticsearch index before rollover")
		listen        = flag.String("listen", ":8080", "TCP network address to listen.")
		statsEnabled  = flag.Bool("stats", true, "Expose Prometheus metrics endpoint")
		versionFlag   = flag.Bool("version", false, "Version")
		debug         = flag.Bool("debug", false, "Debug logging")
	)
	flag.Parse()

	log := logger.NewLogger(*debug)

	if *versionFlag {
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Version:", Version)
		if VersionPrerelease != "" {
			fmt.Println("Version PreRelease:", VersionPrerelease)
		}
		return
	}

	log.Info(fmt.Sprintf("Starting commit: %+v, version: %+v, prerelease: %+v",
		GitCommit, Version, VersionPrerelease))

	if *url == "" {
		log.Fatal("missing url")
	}

	elastic, err := elasticsearch.NewAdapter(
		log,
		elasticsearch.SetEsUrl(*url),
		elasticsearch.SetEsUser(*user),
                elasticsearch.SetEsPwd(*pass),
		elasticsearch.SetEsIndexMaxAge(*indexMaxAge),
		elasticsearch.SetEsIndexMaxDocs(*indexMaxDocs),
		elasticsearch.SetWorkers(*workers),
		elasticsearch.SetBatchCount(*batchCount),
		elasticsearch.SetBatchSize(*batchSize),
		elasticsearch.SetBatchInterval(*batchInterval),
		elasticsearch.SetStats(*statsEnabled),
	)
	if err != nil {
		log.Fatal("Unable to create elasticsearch adapter:", zap.Error(err))
	}
	defer elastic.Close()

	http.Handle("/metrics", prometheus.Handler())

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("Read error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			log.Error("Decode error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			log.Error("Unmarshal error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		elastic.Write(req.Timeseries)
		if err != nil {
			// log.Println("msg", "Error sending samples to remote storage", "err", err, "storage", "num_samples", len(samples))
			log.Error("Error sending samples to remote storage", zap.Error(err))
		}
	})

	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("Read error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			log.Error("Decode error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.ReadRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			log.Error("Unmarshal error", zap.Error(err))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// var resp *prompb.ReadResponse
		resp, err := elastic.Read(req.Queries)
		if err != nil {
			log.Error("Error executing query", zap.String("query", req.String()), zap.Error(err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data, err := proto.Marshal(&prompb.ReadResponse{Results: resp})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-protobuf")
		w.Header().Set("Content-Encoding", "snappy")

		compressed = snappy.Encode(nil, data)
		if _, err := w.Write(compressed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Info(fmt.Sprintf("Listening on %+v", *listen))
	http.ListenAndServe(*listen, nil)
}
