package reorg

import (
	"context"
	"fmt"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
)

const (
	redisKeyBlockHash  = "extractor/reorgwatcher/%s/blockhash"
	redisKeyParentHash = "extractor/reorgwatcher/%s/parenthash"
)

type Service struct {
	redis *redis.Client
}

func NewService(redis *redis.Client) *Service {
	return &Service{
		redis: redis,
	}
}

func (s *Service) readHashesFromRedis(fromBlock int64, toBlock int64, redisKey string) (map[int64]string, error) {
	blocks := make([]string, 0)
	for i := fromBlock; i <= toBlock; i++ {
		blocks = append(blocks, fmt.Sprintf("%d", i))
	}
	hashes := make(map[int64]string)
	values, err := s.redis.HMGet(context.Background(), redisKey, blocks...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		if value != nil {
			hashes[fromBlock+int64(i)] = value.(string)
		}
	}
	return hashes, nil
}

func (s *Service) writeHashToRedis(block int64, redisKey string, hash string) error {
	_, err := s.redis.HSet(context.Background(), redisKey, fmt.Sprintf("%d", block), hash[2:10]).Result()
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DetectReorg(blockchainName string, fromBlock int64, toBlock int64) (bool, error) {
	if fromBlock == 0 {
		// do not check genesis
		fromBlock = 1
	}
	hashes, err := s.readHashesFromRedis(fromBlock-1, toBlock, fmt.Sprintf(redisKeyBlockHash, blockchainName))
	if err != nil {
		return true, err
	}
	parents, err := s.readHashesFromRedis(fromBlock-1, toBlock, fmt.Sprintf(redisKeyParentHash, blockchainName))
	if err != nil {
		return true, err
	}
	for block := fromBlock; block <= toBlock; block++ {
		expectedParent, ok := hashes[block-1]
		if !ok {
			return true, errors.Errorf("failed to check reorg for block %d, expected parent was not found in redis", block)
		}
		parent, ok := parents[block]
		if !ok {
			return true, errors.Errorf("failed to check reorg for block %d, actual parent was not found in redis", block)
		}
		if strings.ToLower(expectedParent) != strings.ToLower(parent) {
			log.Infof("[%s] reorg occured in the block %d: parent 0x%s of %d is not equal to hash 0x%s of %d", blockchainName, block-1, parent, block, expectedParent, block-1)
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Add(blockchainName string, block *proto.BlockChanged) error {
	err := s.writeHashToRedis(int64(block.BlockHeight), fmt.Sprintf(redisKeyParentHash, blockchainName), block.ParentHash)
	if err != nil {
		return err
	}
	err = s.writeHashToRedis(int64(block.BlockHeight), fmt.Sprintf(redisKeyBlockHash, blockchainName), block.BlockHash)
	if err != nil {
		return err
	}
	return nil
}
