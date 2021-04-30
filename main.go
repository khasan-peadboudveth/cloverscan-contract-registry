package main

import (
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/common"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/entity"
	"github.com/clover-network/cloverscan-contract-registry/src/kafka"
	"github.com/clover-network/cloverscan-contract-registry/src/server"
	"github.com/clover-network/cloverscan-contract-registry/src/service"
	"github.com/clover-network/cloverscan-contract-registry/src/service/caller"
	"github.com/clover-network/cloverscan-contract-registry/src/service/reorg"
	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
	"go.uber.org/dig"
)

func main() {
	container := dig.New()

	/* initialize config, emitter and postgresql connection */
	must(container.Provide(config.NewConfig))
	must(container.Provide(kafka.NewProducer))
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
	must(container.Provide(entity.NewConnection))
	must(container.Provide(server.NewGrpcServer))
	must(container.Provide(service.NewService))
	must(container.Provide(reorg.NewService))
	must(container.Provide(caller.NewService))
	must(container.Invoke(func(
		cfg *config.Config,
		kafkaProducer *kafka.Producer,
		mainService *service.Service,
		server *server.GrpcServer,
	) {
		server.Start()
		mainService.Start()
		/* wait for application termination */
		common.WaitForSignal()
	}))
}

func must(err error) {
	if err != nil {
		log.Fatalf("failed to initialize DI: %s", err)
	}
}
