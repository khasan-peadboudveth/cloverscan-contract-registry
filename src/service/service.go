package service

import (
	"fmt"
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	"github.com/clover-network/cloverscan-contract-registry/src/entity"
	"github.com/clover-network/cloverscan-contract-registry/src/nodes"
	proto "github.com/clover-network/cloverscan-proto-contract"
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Service struct {
	database      *pg.DB
	contractCache map[string]*entity.Contract
	mutex         sync.Mutex
	config        *config.Config
	nodesService  *nodes.Service
}

func NewService(database *pg.DB, config *config.Config, nodesService *nodes.Service) *Service {
	return &Service{
		database:      database,
		contractCache: make(map[string]*entity.Contract),
		config:        config,
		nodesService:  nodesService,
	}
}

func metrics(start int64, changes []*proto.BlockChanged) {
	latest := make(map[string]uint64)
	for _, change := range changes {
		if change.BlockHeight > latest[change.BlockchainName] {
			latest[change.BlockchainName] = change.BlockHeight
		}
	}
	delta := float64(time.Now().UnixNano()-start) / 1e9
	tps := int64(float64(len(changes)) / delta)
	log.Infof("batch processed size=%d time=%dms tps=%d latest=%v", len(changes), int64(delta*1000), tps, latest)
}

func (s *Service) Start() {
	go s.scanContracts()
}

func (s *Service) scanContracts() {
	for {
		contact, err := s.nextContract()
		if err != nil {
			if err != pg.ErrNoRows {
				log.Errorf("failed to get next contract from database: %s", err)
				time.Sleep(1 * time.Minute)
			}
			time.Sleep(5 * time.Second)
			continue
		}
		log.Infof("new unprocessed contact was found %s %s", contact.BlockchainName, contact.Address)
		err = s.scanContract(contact)
		if err != nil {
			log.Errorf("failed to scan contract: %s", err)
			time.Sleep(1 * time.Minute)
			continue
		}
	}
}

func (s *Service) scanContract(contract *entity.Contract) error {
	name, err := s.nodesService.GetName(contract.BlockchainName, contract.Address)
	if err != nil {
		return err
	}
	symbol, err := s.nodesService.GetSymbol(contract.BlockchainName, contract.Address)
	if err != nil {
		return err
	}
	decimals, err := s.nodesService.GetDecimals(contract.BlockchainName, contract.Address)
	if err != nil {
		return err
	}
	if decimals == 0 {
		decimals = 18
	}
	contract.Name = name
	contract.Symbol = symbol
	contract.Decimals = int(decimals)
	contract.Status = proto.ContractStatus_SCANNED.String()
	_, err = s.database.Model(contract).Update()
	if err != nil {
		return err
	}
	s.mutex.Lock()
	s.contractCache[contractId(contract.BlockchainName, contract.Address)] = contract
	s.mutex.Unlock()
	log.Infof("smart contract %s %s was scanned: %s %s %d", contract.BlockchainName, contract.Address, name, symbol, decimals)
	return nil
}

func (s *Service) nextContract() (*entity.Contract, error) {
	contract := &entity.Contract{}
	err := s.database.Model(contract).Where("status = ?", proto.ContractStatus_FOUND.String()).Limit(1).Select()
	if err != nil {
		return nil, err
	}
	return contract, nil
}

func (s *Service) HandleBlockChanged(changes []*proto.BlockChanged) error {
	start := time.Now().UnixNano()
	for _, change := range changes {
		switch block := change.Block.(type) {
		case *proto.BlockChanged_EthBlock:
			ethBlock := block.EthBlock
			for _, ethTx := range ethBlock.Txs {
				if ethTx.IsContractCall {
					if err := s.RegisterContract(change.BlockchainName, ethTx.ToAddress); err != nil {
						return err
					}
				}
			}
		default:
			return errors.Errorf("proto block type is unknown %s %s", change.BlockchainName, change.BlockHash)
		}
	}
	metrics(start, changes)
	return nil
}

func contractId(blockchainName, address string) string {
	return fmt.Sprintf("%s:%s", blockchainName, address)
}

func (s *Service) RegisterContract(blockchainName, address string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.contractCache[contractId(blockchainName, address)]
	if ok {
		return nil
	}
	model := &entity.Contract{BlockchainName: blockchainName, Address: address, Status: proto.ContractStatus_FOUND.String()}
	inserted, err := s.database.Model(model).WherePK().SelectOrInsert()
	if inserted {
		s.contractCache[contractId(blockchainName, address)] = model
	}
	return err
}

func (s *Service) Contract(blockchainName, address string) (*entity.Contract, error) {
	contract, ok := s.contractCache[contractId(blockchainName, address)]
	if ok {
		return contract, nil
	}
	model := &entity.Contract{BlockchainName: blockchainName, Address: address}
	err := s.database.Model(model).WherePK().Select()
	if err != nil {
		if err == pg.ErrNoRows {
			return &entity.Contract{}, nil
		}
		return nil, err
	}
	s.mutex.Lock()
	s.contractCache[contractId(contract.BlockchainName, contract.Address)] = contract
	s.mutex.Unlock()
	return model, nil
}
