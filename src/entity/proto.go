package entity

import proto "github.com/clover-network/cloverscan-proto-contract"

func (c *Contract) ToProto() *proto.Contract {
	return &proto.Contract{
		BlockchainName: c.BlockchainName,
		Address:        c.Address,
		Name:           c.Name,
		Decimals:       int32(c.Decimals),
		Symbol:         c.Symbol,
		Status:         proto.ContractStatus(proto.ContractStatus_value[c.Status]),
	}
}
