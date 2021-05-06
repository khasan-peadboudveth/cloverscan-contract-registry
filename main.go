package main

import (
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/common"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/entity"
	"github.com/clover-network/cloverscan-contract-registry/src/kafka"
	"github.com/clover-network/cloverscan-contract-registry/src/nodes"
	"github.com/clover-network/cloverscan-contract-registry/src/server"
	"github.com/clover-network/cloverscan-contract-registry/src/service"
	"github.com/go-redis/redis/v8"
	"github.com/olebedev/emitter"
	log "github.com/sirupsen/logrus"
	"go.uber.org/dig"
)

func main() {
	container := dig.New()

	/* initialize config, emitter and postgresql connection */
	must(container.Provide(config.NewConfig))
	must(container.Provide(func() *emitter.Emitter {
		return emitter.New(10)
	}))
	must(container.Provide(kafka.NewProducer))
	must(container.Provide(kafka.NewConsumer))

	must(container.Invoke(entity.EnsureDatabase))
	must(container.Provide(entity.NewConnection))
	must(container.Provide(service.NewService))
	must(container.Provide(nodes.NewService))
	must(container.Invoke(entity.ApplyDatabaseMigrations))
	must(container.Invoke(common.PrometheusExporter))
	must(container.Provide(func(config *config.Config) *redis.Client {
		redisOpts, err := redis.ParseURL(fmt.Sprintf("redis://%s:%s@%s/", config.RedisUser, config.RedisPassword, config.RedisAddress))
		if err != nil {
			log.Fatalf("unable to parse redis URL: %s", err)
		}
		return redis.NewClient(redisOpts)
	}))

	/* initialize internal services */

	must(container.Provide(server.NewGrpcServer))

	must(container.Invoke(func(
		cfg *config.Config,
		kafkaProducer *kafka.Producer,
		kafkaConsumer *kafka.Consumer,
		server *server.GrpcServer,
		service *service.Service,
		nodesService *nodes.Service,
	) {
		nodesService.Start()
		service.Start()
		server.Start()

		/* wait for application termination */
		kafkaConsumer.Start(service.HandleBlockChanged)
		common.WaitForSignal()
	}))
}

func must(err error) {
	if err != nil {
		log.Fatalf("failed to initialize DI: %s", err)
	}
}
