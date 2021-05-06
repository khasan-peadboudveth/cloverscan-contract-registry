//nolint
//lint:file-ignore U1000 ignore unused code, it's generated
package entity

var Columns = struct {
	Contract struct {
		BlockchainName, Address, Name, Decimals, Symbol, Status string
	}
	SchemaMigration struct {
		ID, Dirty string
	}
}{
	Contract: struct {
		BlockchainName, Address, Name, Decimals, Symbol, Status string
	}{
		BlockchainName: "blockchain_name",
		Address:        "address",
		Name:           "name",
		Decimals:       "decimals",
		Symbol:         "symbol",
		Status:         "status",
	},
	SchemaMigration: struct {
		ID, Dirty string
	}{
		ID:    "version",
		Dirty: "dirty",
	},
}

var Tables = struct {
	Contract struct {
		Name, Alias string
	}
	SchemaMigration struct {
		Name, Alias string
	}
}{
	Contract: struct {
		Name, Alias string
	}{
		Name:  "contract",
		Alias: "t",
	},
	SchemaMigration: struct {
		Name, Alias string
	}{
		Name:  "schema_migrations",
		Alias: "t",
	},
}

type Contract struct {
	tableName struct{} `pg:"contract,alias:t,,discard_unknown_columns"`

	BlockchainName string `pg:"blockchain_name,pk"`
	Address        string `pg:"address,pk"`
	Name           string `pg:"name,use_zero"`
	Decimals       int    `pg:"decimals,use_zero"`
	Symbol         string `pg:"symbol,use_zero"`
	Status         string `pg:"status,use_zero"`
}

type SchemaMigration struct {
	tableName struct{} `pg:"schema_migrations,alias:t,,discard_unknown_columns"`

	ID    int64 `pg:"version,pk"`
	Dirty bool  `pg:"dirty,use_zero"`
}
