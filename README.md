# prometheus-es-adapter

## Overview

A read and write adapter for prometheus persistent storage.

#### Exposed Endpoints

| Port | Path     | Description                                      |
| ---- | -------- | ------------------------------------------------ |
| 8000 | /read    | Prometheus remote read endpoint                  |
| 8000 | /write   | Prometheus remote write endpoint                 |
| 9000 | /metrics | Surface Prometheus metrics                       |
| 9000 | /live    | Http probe endpoint to reflect service liveness  |
| 9000 | /ready   | Http probe endpoint reflecting the connection to and state of the Elasticsearch cluster |

## Config

| Env Variables      | Default               | Description                                                        |
| -----------------  | --------------------- | ------------------------------------------------------------------ |
| ES_URL             | http://localhost:9200 | Elasticsearch URL                                                  |
| ES_USER            |                       | Elasticsearch User                                                 |
| ES_PASSWORD        |                       | Elasticsearch User Password                                        |
| ES_WORKERS         | 1                     | Number of batch workers                                            |
| ES_BATCH_MAX_AGE   | 10                    | Max period in seconds between bulk Elasticsearch insert operations | 
| ES_BATCH_MAX_DOCS  | 1000                  | Max items for bulk Elasticsearch insert operation                  |
| ES_BATCH_MAX_SIZE  | 4096                  | Max size in bytes for bulk Elasticsearch insert operation          |
| ES_ALIAS           | prom-metrics          | Elasticsearch alias pointing to active write index                 |
| ES_INDEX_DAILY     | false                 | Create daily indexes and disable index rollover                    |
| ES_INDEX_SHARDS    | 5                     | Number of Elasticsearch shards to create per index                 |
| ES_INDEX_REPLICAS  | 1                     | Number of Elasticsearch replicas to create per index               |
| ES_INDEX_MAX_AGE   | 7d                    | Max age of Elasticsearch index before rollover                     |
| ES_INDEX_MAX_DOCS  | 1000000               | Max number of docs in Elasticsearch index before rollover          |
| ES_INDEX_MAX_SIZE  |                       | Max size of index before rollover eg 5gb                           |
| ES_SEARCH_MAX_DOCS | 1000                  | Max number of docs returned for Elasticsearch search operation     |
| ES_SNIFF           | false                 | Enable Elasticsearch sniffing                                      |
| STATS              | true                  | Expose Prometheus metrics endpoint                                 |
| DEBUG              | false                 | Display extra debug logs                                           |

## Notes

Although *prometheus-es-adapter* will create and rollover Elasticsearch indicies it is expected that a tool such as Elasticsearch Curator will be used to maintain quiescent indicies eg deleting, shrinking and merging old indexes.

## Requirements

* 6.x Elastisearch cluster

## Getting started

Automated builds of Docker image are available at https://hub.docker.com/r/pwillie/prometheus-es-adapter/.

## Contributing

Local development requires Go to be installed. On OS X with Homebrew you can just run `brew install go`.

Running it then should be as simple as:

```console
$ make build
$ ./bin/prometheus-es-adapter
```

### Testing

`make test`

#### e2e

To run end to end tests using docker-compose, from the "test" directory:
```
docker-compose up -d
docker-compose ps
docker-compose up -d --build prometheus-es-adapter
docker-compose logs -f prometheus-es-adapter
```