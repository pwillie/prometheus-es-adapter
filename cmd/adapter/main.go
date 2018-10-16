package main

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"github.com/TV4/graceful"
	gorilla "github.com/gorilla/handlers"
	"github.com/namsral/flag"
	"github.com/pwillie/prometheus-es-adapter/pkg/elasticsearch"
	"github.com/pwillie/prometheus-es-adapter/pkg/handlers"
	"github.com/pwillie/prometheus-es-adapter/pkg/logger"
	elastic "gopkg.in/olivere/elastic.v6"
)

func main() {
	var (
		url           = flag.String("es_url", "http://localhost:9200", "Elasticsearch URL.")
		user          = flag.String("es_user", "", "Elasticsearch User.")
		pass          = flag.String("es_password", "", "Elasticsearch User Password.")
		workers       = flag.Int("es_workers", 1, "Number of batch workers.")
		batchMaxAge   = flag.Int("es_batch_max_age", 10, "Max period in seconds between bulk Elasticsearch insert operations")
		batchMaxDocs  = flag.Int("es_batch_max_docs", 1000, "Max items for bulk Elasticsearch insert operation")
		batchMaxSize  = flag.Int("es_batch_max_size", 4096, "Max size in bytes for bulk Elasticsearch insert operation")
		indexAlias    = flag.String("es_alias", "prom-metrics", "Elasticsearch alias pointing to active write index")
		indexShards   = flag.Int("es_index_shards", 5, "Number of Elasticsearch shards to create per index")
		indexReplicas = flag.Int("es_index_replicas", 1, "Number of Elasticsearch replicas to create per index")
		indexMaxAge   = flag.String("es_index_max_age", "7d", "Max age of Elasticsearch index before rollover")
		indexMaxDocs  = flag.Int64("es_index_max_docs", 1000000, "Max number of docs in Elasticsearch index before rollover")
		indexMaxSize  = flag.String("es_index_max_size", "", "Max size of index before rollover eg 5gb")
		searchMaxDocs = flag.Int("es_search_max_docs", 1000, "Max number of docs returned for Elasticsearch search operation")
		sniffEnabled  = flag.Bool("es_sniff", false, "Enable Elasticsearch sniffing")
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

	ctx := context.TODO()

	client, err := elastic.NewClient(
		elastic.SetURL(*url),
		elastic.SetBasicAuth(*user, *pass),
		elastic.SetSniff(*sniffEnabled),
	)
	if err != nil {
		log.Fatal("Failed to create elastic client", zap.Error(err))
	}
	defer client.Stop()

	indexCfg := &elasticsearch.IndexConfig{
		Alias:    *indexAlias,
		MaxAge:   *indexMaxAge,
		MaxDocs:  *indexMaxDocs,
		MaxSize:  *indexMaxSize,
		Shards:   *indexShards,
		Replicas: *indexReplicas,
	}
	_, err = elasticsearch.NewIndexService(ctx, log, client, indexCfg)
	if err != nil {
		log.Fatal("Failed to create indexer", zap.Error(err))
	}

	readCfg := &elasticsearch.ReadConfig{
		Alias:   *indexAlias,
		MaxDocs: *searchMaxDocs,
	}
	readSvc := elasticsearch.NewReadService(log, client, readCfg)

	writeCfg := &elasticsearch.WriteConfig{
		Alias:   *indexAlias,
		MaxAge:  *batchMaxAge,
		MaxDocs: *batchMaxDocs,
		MaxSize: *batchMaxSize,
		Workers: *workers,
		Stats:   *statsEnabled,
	}
	writeSvc, err := elasticsearch.NewWriteService(ctx, log, client, writeCfg)
	if err != nil {
		log.Fatal("Unable to create elasticsearch adapter:", zap.Error(err))
	}
	defer writeSvc.Close()

	router := handlers.NewRouter(writeSvc, readSvc)

	graceful.ListenAndServe(&http.Server{
		Addr: ":8000",
		Handler: gorilla.RecoveryHandler(gorilla.PrintRecoveryStack(true))(
			gorilla.CompressHandler(router),
		),
	})
	// TODO: graceful shutdown of bulk processor
}
