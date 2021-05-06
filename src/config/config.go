package config

import (
	"github.com/namsral/flag"
)

type Config struct {
	GrpcAddress       string
	DatabaseAddress   string
	DatabaseUser      string
	DatabasePassword  string
	DatabaseDb        string
	KafkaAddress      string
	KafkaGroup        string
	PrometheusAddress string
	RedisAddress      string
	RedisUser         string
	RedisPassword     string

	GenesisStart bool
}

func NewConfig() *Config {
	config := &Config{}
	/* gRPC */
	flag.StringVar(&config.GrpcAddress, "grpc-address", "localhost:8059", "gRPC address and port for inter-service communications")
	/* PostgreSQL */
	flag.StringVar(&config.DatabaseAddress, "database-address", "localhost:5432", "")
	flag.StringVar(&config.DatabaseUser, "database-user", "postgres", "")
	flag.StringVar(&config.DatabasePassword, "database-password", "12345", "")
	flag.StringVar(&config.DatabaseDb, "database-db", "cloverscan_contract_registry", "")
	/* Apache Kafka */
	flag.StringVar(&config.KafkaAddress, "kafka-address", "localhost:9092", "")
	flag.StringVar(&config.KafkaGroup, "kafka-group", "cloverscan_contract_registry", "")
	flag.StringVar(&config.PrometheusAddress, "prometheus-address", "localhost:8075", "host and port for prometheus")
	/* redis */
	flag.StringVar(&config.RedisAddress, "redis-address", "localhost:6379", "")
	flag.StringVar(&config.RedisUser, "redis-user", "", "")
	flag.StringVar(&config.RedisPassword, "redis-password", "123456", "")

	flag.BoolVar(&config.GenesisStart, "genesis-start", false, "")
	/* parse config from envs or config files */
	flag.Parse()
	return config
}
