package common

import "math/big"

func EthFromGweiUint64(value uint64) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetUint64(value), big.NewFloat(1e9))
}

func EthFromGweiInt64(value int64) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetInt64(value), big.NewFloat(1e9))
}

func EthFromWeiUint64(value uint64) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetUint64(value), big.NewFloat(1e18))
}

func EthFromGweiString(value string) *big.Float {
	floatValue, ok := new(big.Float).SetString(value)
	if !ok {
		return big.NewFloat(0)
	}
	return new(big.Float).Quo(floatValue, big.NewFloat(1e9))
}

func EthFromWeiString(value string) *big.Float {
	floatValue, ok := new(big.Float).SetString(value)
	if !ok {
		return big.NewFloat(0)
	}
	return new(big.Float).Quo(floatValue, big.NewFloat(1e18))
}

func FloatFromStringOrZero(value string) *big.Float {
	floatValue, ok := new(big.Float).SetString(value)
	if !ok {
		return big.NewFloat(0)
	}
	return floatValue
}

func IntFromStringOrZero(value string) *big.Int {
	floatValue, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return big.NewInt(0)
	}
	return floatValue
}
