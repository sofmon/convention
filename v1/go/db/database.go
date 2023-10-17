package db

import (
	_ "github.com/lib/pq"

	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"

	convCfg "github.com/sofmon/convention/v1/go/cfg"
)

type Engine string

const (
	EnginePostgres Engine = "postgres"

	configKeyDatabase convCfg.ConfigKey = "database"
)

type Version string

type connection struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type connections struct {
	Default *connection  `json:"default,omitempty"`
	Shards  []connection `json:"shards,omitempty"`
}

type config struct {
	Engine   Engine                  `json:"engine"`
	Versions map[Version]connections `json:"versions"`
}

func (conn connection) Open() (*sql.DB, error) {
	return sql.Open(
		"postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
			conn.Host, conn.Port, conn.Username, conn.Password, conn.Database,
		),
	)
}

func Open(version Version) (err error) {

	fullCfg, err := convCfg.Object[config](configKeyDatabase)
	if err != nil {
		return
	}

	if fullCfg.Engine != EnginePostgres {
		return fmt.Errorf("database implementation does not support engine '%s'", fullCfg.Engine)
	}

	cfg, ok := fullCfg.Versions[version]
	if !ok {
		return fmt.Errorf("database configuration does not contain the requested configuration version '%s'", version)
	}

	if cfg.Default != nil {
		defaultDB, err = cfg.Default.Open()
		if err != nil {
			return
		}
	}

	for _, shard := range cfg.Shards {
		shardDB, err := shard.Open()
		if err != nil {
			return err
		}
		shardDBs = append(shardDBs, shardDB)
	}

	return
}

func Close() (err error) {
	if defaultDB != nil {
		err = defaultDB.Close()
	}
	for _, shard := range shardDBs {
		err = errors.Join(
			err,
			shard.Close(),
		)
	}
	return
}

var (
	defaultDB *sql.DB
	shardDBs  []*sql.DB
)

var (
	ErrNoDBDefault = errors.New("data is not configured with default database")
	ErrNoDBShards  = errors.New("data is not configured with database shards")
)

func indexByShardKey(key string, count int) int {
	return int(crc32.ChecksumIEEE([]byte(key)) % uint32(count))
}

func Default() *sql.DB {
	if defaultDB == nil {
		panic(ErrNoDBDefault)
	}
	return defaultDB
}

func Shards() []*sql.DB {
	if len(shardDBs) <= 0 {
		panic(ErrNoDBShards)
	}
	return shardDBs
}

func dbByShardKey(key string) *sql.DB {
	if len(shardDBs) <= 0 {
		panic(ErrNoDBShards)
	}
	return shardDBs[indexByShardKey(key, len(shardDBs))]
}

func dbsByShardKeys(keys ...string) (res []*sql.DB) {
	if len(shardDBs) <= 0 {
		panic(ErrNoDBShards)
	}

	sis := map[int]any{}

	for _, key := range keys {
		si := indexByShardKey(key, len(shardDBs))
		sis[int(si)] = nil
	}

	for si := range sis {
		res = append(res, shardDBs[si])
	}
	return
}
