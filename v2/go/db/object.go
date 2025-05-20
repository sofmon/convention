package db

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type Key[idT, shardKeyT ~string] struct {
	ID       idT
	ShardKey shardKeyT
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

func NewObjectSet[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string](vault Vault) ObjectSetSetup[objT, idT, shardKeyT] {

	obj := new(objT)
	objType := reflect.TypeOf(*obj)

	return &objectSet[objT, idT, shardKeyT]{
		vault:   vault,
		objType: objType,
	}
}

func (os *objectSet[objT, idT, shardKeyT]) WithTextSearch() ObjectSetSetup[objT, idT, shardKeyT] {
	os.textSearch = true
	return os
}

func (os *objectSet[objT, idT, shardKeyT]) WithIndexes(indexes ...string) ObjectSetSetup[objT, idT, shardKeyT] {
	os.indexes = append(os.indexes, os.objType.Name())
	return os
}

func (os *objectSet[objT, idT, shardKeyT]) WithCompute(compute func(ctx convCtx.Context, md Metadata, obj *objT) error) ObjectSetSetup[objT, idT, shardKeyT] {
	os.compute = append(os.compute, compute)
	return os
}

func (os *objectSet[objT, idT, shardKeyT]) Ready() ObjectSetReady[objT, idT, shardKeyT] {
	return os
}

type ObjectSetSetup[objT Object[idT, shardKeyT], idT, shardKeyT ~string] interface {
	WithTextSearch() ObjectSetSetup[objT, idT, shardKeyT]
	WithIndexes(indexes ...string) ObjectSetSetup[objT, idT, shardKeyT]
	WithCompute(compute func(ctx convCtx.Context, md Metadata, obj *objT) error) ObjectSetSetup[objT, idT, shardKeyT]
	Ready() ObjectSetReady[objT, idT, shardKeyT]
}

type ObjectSetReady[objT Object[idT, shardKeyT], idT, shardKeyT ~string] interface {
	Tenant(tenant convAuth.Tenant) TenantObjectSet[objT, idT, shardKeyT]
}

type objectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	prepared   bool
	vault      Vault
	textSearch bool
	indexes    []string
	compute    []func(ctx convCtx.Context, md Metadata, obj *objT) error
	objType    reflect.Type
	table      dbTable
}

func (os *objectSet[objT, idT, shardKeyT]) prepare() (err error) {

	if os.prepared {
		return nil
	}

	err = Open()
	if err != nil {
		return
	}

	tenantDBs, ok := dbs[os.vault]
	if !ok {
		err = ErrNoDBVault
		return
	}

	if _, ok := typeToTable[os.vault]; !ok {
		typeToTable[os.vault] = map[reflect.Type]dbTable{}
	}

	os.table, ok = typeToTable[os.vault][os.objType]
	if ok {
		return // object already registered for that vault
	}

	if len(os.indexes) != 0 {
		for _, index := range os.indexes {
			if index == textSearchIndex {
				err = fmt.Errorf("cannot use '%s' as an index field as it is reserved for text search", textSearchIndex)
				return
			}
		}
	}

	runtimeTableName := toSnakeCase(os.objType.Name())
	historyTableName := runtimeTableName + historySuffix
	lockTableName := runtimeTableName + lockSuffix

	createScript := `CREATE TABLE IF NOT EXISTS "` + runtimeTableName + `" (
"id" text PRIMARY KEY,
"created_at" timestamp NOT NULL,
"created_by" text NOT NULL,
"updated_at" timestamp NOT NULL,
"updated_by" text NOT NULL,
"object" JSONB NULL`

	if os.textSearch {
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

	for _, index := range os.indexes {
		createScript += `CREATE INDEX IF NOT EXISTS "` + runtimeTableName + `_` + index + `"
ON "` + runtimeTableName + `" USING gin (("object"->'` + index + `'));
`
	}

	if os.textSearch {
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

	os.table = dbTable{
		ObjectType:       os.objType,
		ObjectTypeName:   os.objType.Name(),
		RuntimeTableName: runtimeTableName,
		HistoryTableName: historyTableName,
		LockTableName:    lockTableName,
		TextSearch:       os.textSearch,
	}

	typeToTable[os.vault][os.objType] = os.table

	return
}

func (os objectSet[objT, idT, shardKeyT]) Tenant(tenant convAuth.Tenant) TenantObjectSet[objT, idT, shardKeyT] {
	return TenantObjectSet[objT, idT, shardKeyT]{
		objectSet: os,
		tenant:    convAuth.Tenant(tenant),
	}
}

type TenantObjectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	objectSet[objT, idT, shardKeyT]
	tenant convAuth.Tenant
}
