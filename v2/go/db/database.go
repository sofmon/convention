package db

import (
	"database/sql"
	"errors"
	"fmt"
	"hash/crc32"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCfg "github.com/sofmon/convention/v2/go/cfg"
)

type Engine string

const (
	EnginePostgres Engine = "postgres"
	EngineSqlite3  Engine = "sqlite3"

	configKeyDatabase convCfg.ConfigKey = "database"
)

type Vault string

type connection struct {
	Engine   Engine `json:"engine"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	InMemory bool   `json:"in_memory"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type connections []connection

func (conn connection) Open() (*sql.DB, error) {
	switch conn.Engine {
	case EnginePostgres:
		return sql.Open(
			"postgres",
			fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
				conn.Host, conn.Port, conn.Username, conn.Password, conn.Database,
			),
		)
	case EngineSqlite3:
		if conn.InMemory {
			return sql.Open("sqlite3", ":memory:")
		} else {
			// TODO: implement file-based sqlite databases
			return nil, errors.New("sqlite engine does not support file-based databases")
		}
	default:
		return nil, fmt.Errorf("unsupported engine: %s", conn.Engine)
	}
}

type config map[Vault]map[convAuth.Tenant]connections

var (
	dbs map[Vault]map[convAuth.Tenant][]*sql.DB

	ErrNoDBTenant = errors.New("db is not configured with provided tenant")
	ErrNoDBVault  = errors.New("db is not configured with provided vault")
)

func Open() (err error) {

	if dbs != nil {
		return
	}

	dbs = make(map[Vault]map[convAuth.Tenant][]*sql.DB)

	cfg, err := convCfg.Object[config](configKeyDatabase)
	if err != nil {
		return
	}

	for vault, vaultCfg := range cfg {
		dbs[vault] = make(map[convAuth.Tenant][]*sql.DB)
		for tenant, tenantCfg := range vaultCfg {
			for _, conn := range tenantCfg {
				db, err := conn.Open()
				if err != nil {
					return err
				}
				dbs[vault][tenant] = append(dbs[vault][tenant], db)
			}
		}
	}

	return
}

func Close() (err error) {
	for _, db := range dbs {
		for _, db := range db {
			for _, db := range db {
				err = errors.Join(
					err,
					db.Close(),
				)
			}
		}
	}
	if err != nil {
		return
	}

	dbs = nil
	return
}

func indexByShardKey(key string, count int) int {
	return int(crc32.ChecksumIEEE([]byte(key)) % uint32(count))
}

func DBs(vault Vault, tenant convAuth.Tenant) ([]*sql.DB, error) {

	err := Open()
	if err != nil {
		return nil, err
	}

	vdb, ok := dbs[vault]
	if !ok {
		return nil, ErrNoDBVault
	}

	tdb, ok := vdb[tenant]
	if !ok {
		return nil, ErrNoDBTenant
	}

	return tdb, nil
}

func dbByIndex(vault Vault, tenant convAuth.Tenant, index int) (*sql.DB, error) {

	dbs, err := DBs(vault, tenant)
	if err != nil {
		return nil, err
	}

	if len(dbs) <= 0 {
		return nil, ErrNoDBTenant
	}

	if index < 0 || index >= len(dbs) {
		return nil, errors.New("database index out of range")
	}

	return dbs[index], nil
}

func dbByShardKey(vault Vault, tenant convAuth.Tenant, key string) (*sql.DB, error) {

	dbs, err := DBs(vault, tenant)
	if err != nil {
		return nil, err
	}

	if len(dbs) <= 0 {
		return nil, ErrNoDBTenant
	}

	if len(dbs) > 1 {
		return dbs[indexByShardKey(key, len(dbs))], nil
	}

	return dbs[0], nil
}

func dbsByShardKeys(vault Vault, tenant convAuth.Tenant, keys ...string) ([]*sql.DB, error) {

	dbs, err := DBs(vault, tenant)
	if err != nil {
		return nil, err
	}

	if len(dbs) <= 0 {
		return nil, ErrNoDBTenant
	}

	if len(keys) == 0 {
		return dbs, nil
	}

	sis := map[int]any{}

	for _, key := range keys {
		sis[indexByShardKey(key, len(dbs))] = nil
	}

	var res []*sql.DB
	for si := range sis {
		res = append(res, dbs[si])
	}

	return res, nil
}
