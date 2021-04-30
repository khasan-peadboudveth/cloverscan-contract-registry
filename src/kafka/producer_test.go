package kafka

import (
	"github.com/clover-network/cloverscan-contract-registry/src/config"
	protocontract "github.com/clover-network/cloverscan-proto-contract"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestProducer_WriteBlock(t *testing.T) {
	cfg := config.NewConfig()
	producer := NewProducer(cfg)
	err := producer.WriteBlockTest(&protocontract.BlockChanged{
		Block: &protocontract.BlockChanged_EthBlock{EthBlock: &protocontract.EthBlock{}},
	})
	require.NoError(t, err)
}

func (producer *Producer) WriteBlockTest(message *protocontract.BlockChanged) error {
	return producer.WriteMessage("v1.test.blockchainextractor.blockchanged", message)
}
