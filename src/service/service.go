package service

import (
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/entity"
	"github.com/clover-network/cloverscan-contract-registry/src/kafka"
	"github.com/clover-network/cloverscan-contract-registry/src/service/caller"
	"github.com/clover-network/cloverscan-contract-registry/src/service/reorg"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

var (
	latestProcessedMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "latest_processed_block"}, []string{"id", "blockchain"})
	latestMetric          = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "latest_block"}, []string{"id", "blockchain"})
)

func init() {
	prometheus.MustRegister(
		latestProcessedMetric,
		latestMetric,
	)
}

type Service struct {
	kafkaProducer *kafka.Producer
	database      *pg.DB
	config        *config.Config
	redis         *redis.Client
	reorgService  *reorg.Service
	callerService *caller.Service
}

func NewService(config *config.Config, kafkaProducer *kafka.Producer, database *pg.DB, redis *redis.Client, reorgService *reorg.Service, callerService *caller.Service) *Service {
	return &Service{
		config:        config,
		kafkaProducer: kafkaProducer,
		database:      database,
		redis:         redis,
		reorgService:  reorgService,
		callerService: callerService,
	}
}

func (s *Service) GetAllActiveExtractorConfigs() ([]*entity.ExtractorConfig, error) {
	var result []*entity.ExtractorConfig
	if err := s.database.Model(&result).Where("enabled = true AND parallel >= 1").Select(); err != nil {
		return []*entity.ExtractorConfig{}, errors.Wrapf(err, "failed to read extractor configurations")
	}
	return result, nil
}

func (s *Service) Start() {
	extractors, err := s.GetAllActiveExtractorConfigs()
	log.Infof("going to start %d extractors", len(extractors))
	if err != nil {
		log.Fatalf("failed to obtain active extractors: %s", err)
	}
	if len(extractors) == 0 {
		log.Fatalf("no active extractors were found, shutting down...")
	}
	for _, extractor := range extractors {
		go s.StartExtractor(extractor)
	}
}

func (s *Service) UpdateLatestProcessedBlock(config *entity.ExtractorConfig, newValue int64) error {
	config.LatestProcessedBlock = newValue
	_, err := s.database.Model(config).WherePK().Column("latest_processed_block").Update()
	return err

}

func (s *Service) StartExtractor(extractorConfig *entity.ExtractorConfig) {
	blockchainName := extractorConfig.Name
	latestProcessedBlock := extractorConfig.LatestProcessedBlock
	oneByOne := 0
	log.Infof("[%s] starting extractor: node: %s, latest processed block: %d...", blockchainName, extractorConfig.Nodes[0], latestProcessedBlock)

	for _, node := range extractorConfig.Nodes {
		err := s.callerService.RegisterNode(blockchainName, node)
		if err != nil {
			log.Errorf("failed to register %s %s: %s", blockchainName, node, err)
		}
	}
	if !s.config.GenesisStart && latestProcessedBlock != -1 {
		// if we allow extractor to start not from genesis block, than we should mark some block start of canonical chain
		protoBlock, err := s.callerService.ProcessBlock(blockchainName, latestProcessedBlock)
		if err != nil {
			log.Errorf("[%s] failed to read latest block #%d to mark it as block from canonical chain: %s", blockchainName, latestProcessedBlock, err)
			return
		}
		err = s.reorgService.Add(blockchainName, protoBlock)
		if err != nil {
			log.Errorf("[%s] failed to mark block #%d to mark it as block from canonical chain: %s", blockchainName, latestProcessedBlock, err)
			return
		}
		log.Infof("[%s] marked block #%d (0x%s) as origin of canonical chain", blockchainName, latestProcessedBlock, protoBlock.BlockHash[2:10])
	}
	for {
		latestBlock, err := s.callerService.LatestBlock(blockchainName)
		if err != nil {
			log.Errorf("[%s] failed to read latest block while on the block %d: %s", blockchainName, latestProcessedBlock, err)
			time.Sleep(time.Second * 5)
			continue
		}
		latestMetric.WithLabelValues(fmt.Sprintf("%d", extractorConfig.ID), blockchainName).Set(float64(latestBlock))
		if latestProcessedBlock >= latestBlock {
			time.Sleep(time.Second * 5)
			continue
		}
		for {
			if latestProcessedBlock >= latestBlock {
				break
			}
			parallelProcessCount := extractorConfig.Parallel
			if int64(parallelProcessCount) > latestBlock-latestProcessedBlock {
				parallelProcessCount = int(latestBlock - latestProcessedBlock)
			}
			if oneByOne > 0 {
				parallelProcessCount = 1
			}
			processFrom := latestProcessedBlock + 1
			processTo := latestProcessedBlock + int64(parallelProcessCount)
			var wg sync.WaitGroup
			wg.Add(parallelProcessCount)
			errorOccurred := false
			blocks := make([]*proto.BlockChanged, 0)
			for i := 0; i < parallelProcessCount; i++ {
				block := latestProcessedBlock + int64(i) + 1
				go func() {
					defer wg.Done()
					protoBlock, processErr := s.callerService.ProcessBlock(blockchainName, block)
					if processErr != nil {
						log.Errorf("[%s] failed to process block %d: %s", blockchainName, block, processErr)
						errorOccurred = true
						return
					}
					blocks = append(blocks, protoBlock)
					reorgAddErr := s.reorgService.Add(blockchainName, protoBlock)
					if reorgAddErr != nil {
						log.Errorf("[%s] failed to add block %d to reorg service: %s", blockchainName, block, processErr)
						errorOccurred = true
						return
					}
				}()
			}
			wg.Wait()
			if errorOccurred {
				oneByOne = parallelProcessCount
				break
			}

			reorgOccurred, err := s.reorgService.DetectReorg(blockchainName, processFrom, processTo)
			if err != nil {
				log.Errorf("[%s] failed to check for reorgs for blocks %d-%d: %s", blockchainName, processFrom, processTo, err)
				break
			}
			if reorgOccurred {
				latestProcessedBlock = latestProcessedBlock - int64(parallelProcessCount)
				if latestProcessedBlock < -1 {
					latestProcessedBlock = -1
				}
				break
			}

			err = s.kafkaProducer.WriteBlock(blocks...)
			if err != nil {
				log.Errorf("[%s] failed to write blocks to kafka: %s", blockchainName, err)
				break
			}

			latestProcessedBlock = processTo
			err = s.UpdateLatestProcessedBlock(extractorConfig, latestProcessedBlock)
			if err != nil {
				log.Errorf("[%s] failed to update latest processed block %d: %s", blockchainName, latestProcessedBlock, err)
				break
			}
			latestProcessedMetric.WithLabelValues(fmt.Sprintf("%d", extractorConfig.ID), blockchainName).Set(float64(latestProcessedBlock))

			if oneByOne > 0 {
				oneByOne -= 1
			}

			log.Infof("[%s] processed blocks (%d-%d)/%d", blockchainName, processFrom, processTo, latestBlock)
		}
		time.Sleep(time.Second * 5)
	}
}
