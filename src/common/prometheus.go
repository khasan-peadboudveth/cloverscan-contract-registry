package common

import (
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func PrometheusExporter(config *config.Config) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(config.PrometheusAddress, nil)
		if err != nil {
			log.Fatalf("failed to start prometheus endpoint: %s", err)
		}
		log.Printf("prometheus exporter is listening on address %s", config.PrometheusAddress)
	}()
}
