# prometheus-es-adapter

## Overview

A read and write adapter for prometheus persistent storage

## Types

| Env Variables      | Default               | Description                                                        |
| -----------------  | --------------------- | ------------------------------------------------------------------ |
| ES_URL             | http://localhost:9200 | Elasticsearch URL                                                  |
| ES_USER            |                       | Elasticsearch User                                                 |
| ES_PASSWORD        |                       | Elasticsearch User Password                                        |
| ES_WORKERS         | 0                     | Number of batch workers                                            |
| ES_BATCH_COUNT     | 1000                  | Max items for bulk Elasticsearch insert operation                  |
| ES_BATCH_SIZE      | 4096                  | Max size in bytes for bulk Elasticsearch insert operation          |
| ES_BATCH_INTERVAL  | 10                    | Max period in seconds between bulk Elasticsearch insert operations |
| ES_INDEX_MAX_AGE   | 7d                    | Max age of Elasticsearch index before rollover                     |
| ES_INDEX_MAX_DOCS  | 1000000               | Max documents in Elasticsearch index before rollover               |
| ES_SEARCH_MAX_DOCS | 1000                  | Max documents returned by Elasticsearch search operations          |
| ES_SNIFF           | false                 | Enable Elasticsearch sniffing                                      |
| STATS              | true                  | Expose Prometheus metrics endpoint                                 |
| LISTEN             | :8080                 | TCP network address to start http listener on                      |
| VERSION            |                       | Display version and exit                                           |

## Requirements

* 5.x or 6.x Elastisearch cluster

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
