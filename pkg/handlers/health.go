package handlers

import (
	"errors"
	"net/http"

	"github.com/heptiolabs/healthcheck"
	elastic "gopkg.in/olivere/elastic.v6"
)

func healthzHandler(client *elastic.Client) http.Handler {
	health := healthcheck.NewHandler()
	health.AddReadinessCheck("elasticsearch",
		esCheck(client))
	return health
}

func esCheck(client *elastic.Client) healthcheck.Check {
	return func() error {
		if client == nil {
			return errors.New("Elasticsearch client is nil")
		}
		if err := client.WaitForGreenStatus("1s"); err != nil {
			if err := client.WaitForYellowStatus("1s"); err != nil {
				return err
			}
		}
		return nil
	}
}
