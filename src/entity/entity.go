//nolint
//lint:file-ignore U1000 ignore unused code, it's generated
package entity

var Columns = struct {
	ExtractorConfig struct {
		ID, Name, ChainType, LatestProcessedBlock, Enabled, Parallel, Nodes string
	}
	SchemaMigration struct {
		ID, Dirty string
	}
}{
	ExtractorConfig: struct {
		ID, Name, ChainType, LatestProcessedBlock, Enabled, Parallel, Nodes string
	}{
		ID:                   "id",
		Name:                 "name",
		ChainType:            "chain_type",
		LatestProcessedBlock: "latest_processed_block",
		Enabled:              "enabled",
		Parallel:             "parallel",
		Nodes:                "nodes",
	},
	SchemaMigration: struct {
		ID, Dirty string
	}{
		ID:    "version",
		Dirty: "dirty",
	},
}

var Tables = struct {
	ExtractorConfig struct {
		Name, Alias string
	}
	SchemaMigration struct {
		Name, Alias string
	}
}{
	ExtractorConfig: struct {
		Name, Alias string
	}{
		Name:  "extractor_config",
		Alias: "t",
	},
	SchemaMigration: struct {
		Name, Alias string
	}{
		Name:  "schema_migrations",
		Alias: "t",
	},
}

type ExtractorConfig struct {
	tableName struct{} `pg:"extractor_config,alias:t,,discard_unknown_columns"`

	ID                   int      `pg:"id,pk"`
	Name                 string   `pg:"name,use_zero"`
	ChainType            string   `pg:"chain_type,use_zero"`
	LatestProcessedBlock int64    `pg:"latest_processed_block,use_zero"`
	Enabled              bool     `pg:"enabled,use_zero"`
	Parallel             int      `pg:"parallel,use_zero"`
	Nodes                []string `pg:"nodes,array,use_zero"`
}

type SchemaMigration struct {
	tableName struct{} `pg:"schema_migrations,alias:t,,discard_unknown_columns"`

	ID    int64 `pg:"version,pk"`
	Dirty bool  `pg:"dirty,use_zero"`
}
