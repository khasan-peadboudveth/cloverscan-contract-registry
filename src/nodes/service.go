package nodes

import (
	"context"
	"encoding/json"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-redis/redis/v8"
	"github.com/olebedev/emitter"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/big"
	"sync"
)

const RedisNodeConfigKey = "contractregistry/nodeconfig"

type Service struct {
	redis      *redis.Client
	emitter    *emitter.Emitter
	ethClients map[string]*ethclient.Client
	mutex      sync.Mutex
}

func NewService(redis *redis.Client, emitter *emitter.Emitter) *Service {
	return &Service{
		redis:      redis,
		emitter:    emitter,
		ethClients: make(map[string]*ethclient.Client),
	}

}

func (s *Service) Start() {
	_, err := s.GetAllNodeConfigs()
	if err != nil {
		log.Fatalf("failed to read configs from redis: %s", err)
	}
	go s.listenForEvents()
}

func (s *Service) GetEthClient(blockchainName string) (*ethclient.Client, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	client, ok := s.ethClients[blockchainName]
	if ok {
		return client, nil
	}
	nodeConfig, err := s.GetNodeConfig(blockchainName)
	if err != nil {
		return nil, err
	}
	client, err = ethclient.Dial(nodeConfig.NodeUrl)
	if err != nil {
		return nil, err
	}
	s.ethClients[blockchainName] = client
	return client, nil
}

func (s *Service) callEthContract(blockchainName string, address string, data []byte) ([]byte, error) {
	client, err := s.GetEthClient(blockchainName)
	if err != nil {
		return nil, err
	}
	addr := common.HexToAddress(address)
	return client.CallContract(context.Background(), ethereum.CallMsg{
		From:     common.Address{},
		To:       &addr,
		Gas:      0,
		GasPrice: nil,
		Value:    nil,
		Data:     data,
	}, nil)
}

func (s *Service) GetSymbol(blockchainName string, address string) (string, error) {
	output, err := s.callEthContract(blockchainName, address, crypto.Keccak256([]byte("symbol()")))
	if err != nil {
		return "", err
	}
	return parseEthString(output)
}

func (s *Service) GetName(blockchainName string, address string) (string, error) {
	output, err := s.callEthContract(blockchainName, address, crypto.Keccak256([]byte("name()")))
	if err != nil {
		return "", err
	}
	return parseEthString(output)
}

func (s *Service) GetDecimals(blockchainName string, address string) (int32, error) {
	output, err := s.callEthContract(blockchainName, address, crypto.Keccak256([]byte("decimals()")))
	if err != nil {
		return 0, err
	}
	return int32(new(big.Int).SetBytes(output).Uint64()), nil
}

func parseEthString(output []byte) (string, error) {
	if len(output) == 0 {
		return "", nil
	}
	stringType, err := abi.NewType("string", "", nil)
	if err != nil {
		return "", err
	}
	arguments := abi.Arguments{
		abi.Argument{Type: stringType},
	}
	unpacked, err := arguments.Unpack(output)
	if err != nil {
		return "", err
	}
	if len(unpacked) == 0 {
		return "", errors.Errorf("unpacked eth output has 0 length")
	}
	return unpacked[0].(string), nil
}

func (s *Service) listenForEvents() {
	nodeChanged := s.emitter.On("kafkaConsumer/nodesChanged")
	for {
		var err error
		select {
		case event := <-nodeChanged:
			for _, nodeConfig := range event.Args[0].(*proto.NodeChanged).Configs {
				err = s.AddNodeConfig(nodeConfig)
				if err != nil {
					log.Errorf("failed to process node config changed event: %s", err)
				}
			}
		}
	}
}

func (s *Service) AddNodeConfig(nodeConfig *proto.NodeConfig) error {
	bytes, err := json.Marshal(nodeConfig)
	if err != nil {
		return err
	}
	if err := s.redis.HSet(context.Background(), RedisNodeConfigKey, nodeConfig.BlockchainName, bytes).Err(); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetAllNodeConfigs() ([]*proto.NodeConfig, error) {
	values, err := s.redis.HGetAll(context.Background(), RedisNodeConfigKey).Result()
	if err != nil {
		return nil, err
	}
	nodeConfigs := make([]*proto.NodeConfig, 0, len(values))
	for _, value := range values {
		nodeConfig := &proto.NodeConfig{}
		if err := json.Unmarshal([]byte(value), nodeConfig); err != nil {
			return nil, err
		}
		nodeConfigs = append(nodeConfigs, nodeConfig)
	}
	return nodeConfigs, nil
}

func (s *Service) GetNodeConfig(blockchainName string) (*proto.NodeConfig, error) {
	value, err := s.redis.HGet(context.Background(), RedisNodeConfigKey, blockchainName).Result()
	if err != nil {
		return nil, err
	}
	nodeConfig := &proto.NodeConfig{}
	if err := json.Unmarshal([]byte(value), nodeConfig); err != nil {
		return nil, err
	}
	return nodeConfig, nil
}
