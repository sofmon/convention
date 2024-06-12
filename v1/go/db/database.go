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

type Tenant string

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
	Versions map[Version]struct {
		Engine  Engine                 `json:"engine"`
		Tenants map[Tenant]connections `json:"tenants"`
	} `json:"versions"`
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

	versionCfg, ok := fullCfg.Versions[version]
	if !ok {
		return fmt.Errorf("database configuration does not contain the requested configuration version '%s'", version)
	}

	if versionCfg.Engine != EnginePostgres {
		return fmt.Errorf("database implementation does not support engine '%s'", versionCfg.Engine)
	}

	for tenant, tenantCfg := range versionCfg.Tenants {

		var tdb tenantDB

		if tenantCfg.Default != nil {
			tdb.Default, err = tenantCfg.Default.Open()
			if err != nil {
				return
			}
		}

		for _, shard := range tenantCfg.Shards {
			shardDB, err := shard.Open()
			if err != nil {
				return err
			}
			tdb.Shards = append(tdb.Shards, shardDB)
		}

		dbs[tenant] = tdb
	}

	return
}

func Close() (err error) {
	for _, db := range dbs {
		if db.Default != nil {
			err = db.Default.Close()
		}
		for _, shard := range db.Shards {
			err = errors.Join(
				err,
				shard.Close(),
			)
		}
	}
	return
}

type tenantDB struct {
	Default *sql.DB
	Shards  []*sql.DB
}

var (
	dbs = map[Tenant]tenantDB{}
)

var (
	ErrNoDBTenant  = errors.New("data is not configured with provided tenant")
	ErrNoDBDefault = errors.New("data is not configured with default database")
	ErrNoDBShards  = errors.New("data is not configured with database shards")
)

func indexByShardKey(key string, count int) int {
	return int(crc32.ChecksumIEEE([]byte(key)) % uint32(count))
}

func Default(tenant Tenant) *sql.DB {

	tdb, ok := dbs[tenant]
	if !ok {
		panic(ErrNoDBTenant)
	}

	if tdb.Default == nil {
		panic(ErrNoDBDefault)
	}

	return tdb.Default
}

func Shards(tenant Tenant) []*sql.DB {

	tdb, ok := dbs[tenant]
	if !ok {
		panic(ErrNoDBTenant)
	}

	if !ok || len(tdb.Shards) <= 0 {
		panic(ErrNoDBShards)
	}
	return tdb.Shards
}

func dbByShardKey(tenant Tenant, key string) *sql.DB {

	tdb, ok := dbs[tenant]
	if !ok {
		panic(ErrNoDBTenant)
	}

	if len(tdb.Shards) <= 0 {
		panic(ErrNoDBShards)
	}
	return tdb.Shards[indexByShardKey(key, len(tdb.Shards))]
}

func dbsByShardKeys(tenant Tenant, keys ...string) (res []*sql.DB) {

	tdb, ok := dbs[tenant]
	if !ok {
		panic(ErrNoDBTenant)
	}

	if len(tdb.Shards) <= 0 {
		panic(ErrNoDBShards)
	}

	sis := map[int]any{}

	for _, key := range keys {
		si := indexByShardKey(key, len(tdb.Shards))
		sis[si] = nil
	}

	for si := range sis {
		res = append(res, tdb.Shards[si])
	}
	return
}
