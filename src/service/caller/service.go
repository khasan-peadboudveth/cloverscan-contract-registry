package caller

import (
	"github.com/clover-network/cloverscan-contract-registry/src/blockchain/eth"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type Caller struct {
	service BlockchainService
	nodeUrl string
	skip    int
}

type Service struct {
	callers map[string][]*Caller
}

const skipAfterFail = 30

type BlockchainService interface {
	ProcessBlock(blockNumber int64) (*proto.BlockChanged, error)
	LatestBlock() (int64, error)
	TransactionByHash(hash string) (*proto.TransactionDetails, error)
	Start() error
}

func NewService() *Service {
	return &Service{
		callers: make(map[string][]*Caller),
	}
}

func ConnectBlockchainService(blockchainName string, nodeUrl string) (BlockchainService, error) {
	switch blockchainName {
	case "ETH", "CLV", "BSC":
		service := BlockchainService(eth.NewService(blockchainName, nodeUrl))
		err := service.Start()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to start blockchain service")
		}
		log.Infof("[%s] node %s was started", blockchainName, nodeUrl)
		return service, nil
		//case "DOT":
	}
	return nil, errors.Errorf("failed to get blockchain service by name %s: no such blockchain service", blockchainName) // exited
}

func (s *Service) RegisterNode(blockchainName string, nodeUrl string) error {
	service, err := ConnectBlockchainService(blockchainName, nodeUrl)
	if err != nil {
		return err
	}
	s.callers[blockchainName] = append(s.callers[blockchainName], &Caller{
		service: service,
		nodeUrl: nodeUrl,
		skip:    0,
	})
	return nil
}

func (s *Service) chooseCaller(blockchainName string) (*Caller, error) {
	callers, ok := s.callers[blockchainName]
	if !ok || len(callers) == 0 {
		return nil, errors.Errorf("no connection for blockchain %s was found", blockchainName)
	}
	chosen := callers[0]
	for _, caller := range callers {
		if caller.skip < chosen.skip {
			chosen = caller
		}
	}
	return chosen, nil
}

func (s *Service) reduceSkip(blockchainName string) {
	for _, caller := range s.callers[blockchainName] {
		if caller.skip > 0 {
			caller.skip -= 1
		}
	}
}

func (s *Service) LatestBlock(blockchainName string) (int64, error) {
	caller, err := s.chooseCaller(blockchainName)
	if err != nil {
		return 0, err
	}
	result, err := caller.service.LatestBlock()
	if err != nil {
		caller.skip = skipAfterFail
		return result, err
	}
	s.reduceSkip(blockchainName)
	return result, nil
}

func (s *Service) ProcessBlock(blockchainName string, blockNumber int64) (*proto.BlockChanged, error) {
	caller, err := s.chooseCaller(blockchainName)
	if err != nil {
		return nil, err
	}
	result, err := caller.service.ProcessBlock(blockNumber)
	if err != nil {
		caller.skip = skipAfterFail
		return result, errors.Wrapf(err, "process block on node %s failed", caller.nodeUrl)
	}
	s.reduceSkip(blockchainName)
	return result, nil
}
