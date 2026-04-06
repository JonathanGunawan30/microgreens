package config

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
)

func (cfg *Config) NewElasticsearchClient() (*elasticsearch.TypedClient, error) {
	configElastic := elasticsearch.Config{
		Addresses: []string{cfg.Elasticsearch.Host},
		Username:  cfg.Elasticsearch.Username,
		Password:  cfg.Elasticsearch.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	es, err := elasticsearch.NewTypedClient(configElastic)
	if err != nil {
		return nil, fmt.Errorf("error creating elastic client: %v", err)
	}

	_, err = es.Info().Do(nil)
	if err != nil {
		return nil, fmt.Errorf("error connecting to elasticsearch: %v", err)
	}

	return es, nil
}
