package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"github.com/pwillie/prometheus-es-adapter/lib/elasticsearch"
)

// Main entry point.
func main() {
	var (
		url         = flag.String("esurl", "http://localhost:9200", "Elasticsearch URL.")
		index       = flag.String("esindex", "prom_storage", "Index name.")
		listen      = flag.String("listen", ":8080", "TCP network address to listen.")
		workers     = flag.Int("workers", 0, "Number of batch workers.")
		versionFlag = flag.Bool("version", false, "Version")
		// debug       = flag.Bool("debug", false, "Debug logging")
	)
	flag.Parse()

	if *versionFlag {
		fmt.Println("Git Commit:", GitCommit)
		fmt.Println("Version:", Version)
		if VersionPrerelease != "" {
			fmt.Println("Version PreRelease:", VersionPrerelease)
		}
		return
	}

	if *url == "" {
		log.Fatal("missing url")
	}
	if *index == "" {
		log.Fatal("missing index name")
	}

	elastic, err := elasticsearch.NewAdapter(
		elasticsearch.SetEsUrl(*url),
		elasticsearch.SetEsIndex(*index),
		elasticsearch.SetWorkers(*workers),
	)
	if err != nil {
		log.Println("Unable to create elasticsearch adapter:", err.Error())
		return
	}
	defer elastic.Close()

	http.Handle("/metrics", prometheus.Handler())

	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("msg", "Read error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			log.Println("msg", "Decode error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.WriteRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			log.Println("msg", "Unmarshal error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		elastic.Write(req.Timeseries)
		if err != nil {
			// log.Println("msg", "Error sending samples to remote storage", "err", err, "storage", "num_samples", len(samples))
			log.Println("msg", "Error sending samples to remote storage", "err", err, "storage", "num_samples")
		}
	})

	http.HandleFunc("/read", func(w http.ResponseWriter, r *http.Request) {
		compressed, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("msg", "Read error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reqBuf, err := snappy.Decode(nil, compressed)
		if err != nil {
			log.Println("msg", "Decode error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req prompb.ReadRequest
		if err := proto.Unmarshal(reqBuf, &req); err != nil {
			log.Println("msg", "Unmarshal error", "err", err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// var resp *prompb.ReadResponse
		resp, err := elastic.Read(req.Queries)
		if err != nil {
			log.Println("msg", "Error executing query", "query", req, "storage", "err", err)
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

	http.ListenAndServe(*listen, nil)
}
