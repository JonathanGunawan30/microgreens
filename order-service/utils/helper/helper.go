package helper

import (
	"order-service/config"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/labstack/gommon/log"
)

func RetryElasticsearch(cfg *config.Config, onReady func(esClient *elasticsearch.TypedClient)) {
	baseDelay := 5 * time.Second
	maxDelay := 12 * time.Hour
	attempt := 0

	for {
		esClient, err := cfg.NewElasticsearchClient()
		if err == nil {
			log.Info("Elasticsearch connected, spawning ES consumers...")
			onReady(esClient)
			return
		}

		attempt++
		delay := baseDelay * time.Duration(1<<attempt)
		if delay > maxDelay {
			delay = maxDelay
		}
		log.Warnf("Elasticsearch not ready (attempt %d), retrying in %v: %v", attempt, delay, err)
		time.Sleep(delay)
	}
}
