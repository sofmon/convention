package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	convAuth "github.com/sofmon/convention/v2/go/auth"
)

type Key[idT, shardKeyT ~string] struct {
	ID       idT
	ShardKey shardKeyT
}

type Metadata struct {
	CreatedAt time.Time
	CreatedBy convAuth.User
	UpdatedAt time.Time
	UpdatedBy convAuth.User
}

type Object[idT, shardKeyT ~string] interface {
	DBKey() Key[idT, shardKeyT]
}

type dbTable struct {
	ObjectType       reflect.Type
	ObjectTypeName   string
	RuntimeTableName string
	HistoryTableName string
	LockTableName    string
	TextSearch       bool
}

const (
	historySuffix = "_history"
	lockSuffix    = "_lock"

	textSearchIndex = "text_search"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")

	typeToTable = map[Vault]map[reflect.Type]dbTable{}

	ErrObjectTypeNotRegistered = errors.New("object type not registered - use NewObjectSet to access vaults")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func dbsForShardKeys[shardKeyT ~string](vault Vault, tenant convAuth.Tenant, sks ...shardKeyT) ([]*sql.DB, error) {
	s := make([]string, len(sks))
	for i, sk := range sks {
		s[i] = string(sk)
	}
	return dbsByShardKeys(vault, tenant, s...)
}

func registerObject[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string](vault Vault, textSearch bool, indexes ...string) (table dbTable, err error) {

	tenantDBs, ok := dbs[vault]
	if !ok {
		err = ErrNoDBVault
		return
	}

	if _, ok := typeToTable[vault]; !ok {
		typeToTable[vault] = map[reflect.Type]dbTable{}
	}

	obj := new(objT)
	objType := reflect.TypeOf(*obj)
	objTypeName := objType.Name()

	table, ok = typeToTable[vault][objType]
	if ok {
		return // object already registered for that vault
	}

	if len(indexes) != 0 {
		for _, index := range indexes {
			if index == textSearchIndex {
				err = fmt.Errorf("cannot use '%s' as an index field as it is reserved for text search", textSearchIndex)
				return
			}
		}
	}

	runtimeTableName := toSnakeCase(objType.Name())
	historyTableName := runtimeTableName + historySuffix
	lockTableName := runtimeTableName + lockSuffix

	createScript := `CREATE TABLE IF NOT EXISTS "` + runtimeTableName + `" (
"id" text PRIMARY KEY,
"created_at" timestamp NOT NULL,
"created_by" text NOT NULL,
"updated_at" timestamp NOT NULL,
"updated_by" text NOT NULL,
"object" JSONB NULL`

	if textSearch {
		createScript += `,
"text_search" tsvector GENERATED ALWAYS AS (jsonb_to_tsvector('english', "object", '["all"]')) STORED`
	}

	createScript += `
);
CREATE TABLE IF NOT EXISTS "` + historyTableName + `" (
"id" text NOT NULL,
"created_at" timestamp NOT NULL,
"created_by" text NOT NULL,
"updated_at" timestamp NOT NULL,
"updated_by" text NOT NULL,
"object" JSONB NULL
);
CREATE TABLE IF NOT EXISTS "` + lockTableName + `" (
"id" text PRIMARY KEY,
"created_at" timestamp NOT NULL,
"description" text NOT NULL
);
`

	for _, index := range indexes {
		createScript += `CREATE INDEX IF NOT EXISTS "` + runtimeTableName + `_` + index + `"
ON "` + runtimeTableName + `" USING gin (("object"->'` + index + `'));
`
	}

	if textSearch {
		createScript += `CREATE INDEX IF NOT EXISTS "` + runtimeTableName + `_` + textSearchIndex + `"
ON "` + runtimeTableName + `" USING gin ("text_search");
`
	}

	for _, dbs := range tenantDBs {
		for _, db := range dbs {
			_, err = db.Exec(createScript)
			if err != nil {
				return
			}
		}
	}

	table = dbTable{
		ObjectType:       objType,
		ObjectTypeName:   objTypeName,
		RuntimeTableName: runtimeTableName,
		HistoryTableName: historyTableName,
		LockTableName:    lockTableName,
		TextSearch:       textSearch,
	}

	typeToTable[vault][objType] = table

	return
}

func NewObjectSet[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string](vault Vault, textSearch bool, indexes ...string) (objSet ObjectSet[objT, idT, shardKeyT], err error) {

	table, err := registerObject[objT](vault, textSearch, indexes...)
	if err != nil {
		return
	}

	obj := new(objT)
	objType := reflect.TypeOf(*obj)

	objSet = ObjectSet[objT, idT, shardKeyT]{
		vault:   vault,
		objType: objType,
		table:   table,
	}

	return
}

type ObjectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	vault   Vault
	objType reflect.Type
	table   dbTable
}

func (os ObjectSet[objT, idT, shardKeyT]) isInitialized() bool {
	return os.vault != "" && os.table.ObjectType != nil && os.table.RuntimeTableName != ""
}

func (os ObjectSet[objT, idT, shardKeyT]) Tenant(tenant convAuth.Tenant) TenantObjectSet[objT, idT, shardKeyT] {
	return TenantObjectSet[objT, idT, shardKeyT]{
		ObjectSet: os,
		tenant:    convAuth.Tenant(tenant),
	}
}

type TenantObjectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	ObjectSet[objT, idT, shardKeyT]
	tenant convAuth.Tenant
}
