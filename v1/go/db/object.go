package db

import (
	"database/sql"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type Trail[idT, shardKeyT ~string] struct {
	ID        idT
	ShardKey  shardKeyT
	CreatedAt time.Time
	CreatedBy string
	UpdatedAt time.Time
	UpdatedBy string
}

type Object[idT, shardKeyT ~string] interface {
	Trail() Trail[idT, shardKeyT]
}

type dbTable struct {
	ObjectType       reflect.Type
	ObjectTypeName   string
	RuntimeTableName string
	HistoryTableName string
	LockTableName    string
	Sharding         bool
}

const (
	historySuffix = "_history"
	lockSuffix    = "_lock"
)

var (
	matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")

	typeToTable = map[reflect.Type]dbTable{}

	ErrObjectTypeNotRegistered = errors.New("object type not registered - use RegisterObject before using specific type")
	ErrObjectNotUsingShards    = errors.New("object not using shards while shards are supplied to query")
)

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func dbsForShardKeys[shardKeyT ~string](tenant Tenant, sks ...shardKeyT) []*sql.DB {

	switch len(sks) {

	case 1:
		return []*sql.DB{dbByShardKey(tenant, string(sks[0]))}

	case 0:
		return Shards(tenant)

	default:
		sksStr := make([]string, len(sks))
		for i, sk := range sks {
			sksStr[i] = string(sk)
		}
		return dbsByShardKeys(tenant, sksStr...)

	}

}

func RegisterObject[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string](sharding bool, indexes ...string) (err error) {

	obj := new(objT)
	objType := reflect.TypeOf(*obj)
	objTypeName := objType.Name()

	runtimeTableName := toSnakeCase(objType.Name())
	historyTableName := runtimeTableName + historySuffix
	lockTableName := runtimeTableName + lockSuffix

	createScript := `CREATE TABLE IF NOT EXISTS "` + runtimeTableName + `" (
"id" text PRIMARY KEY,
"created_at" timestamp DEFAULT now(),
"created_by" text NOT NULL,
"updated_at" timestamp DEFAULT now(),
"updated_by" text NOT NULL,
"object" JSONB NULL
);
CREATE TABLE IF NOT EXISTS "` + historyTableName + `" (
"id" text NOT NULL,
"created_at" timestamp DEFAULT now(),
"created_by" text NOT NULL,
"updated_at" timestamp DEFAULT now(),
"updated_by" text NOT NULL,
"object" JSONB NULL
);
CREATE TABLE IF NOT EXISTS "` + lockTableName + `" (
"id" text PRIMARY KEY,
"created_at" timestamp DEFAULT now(),
"description" text NOT NULL
);
`

	for _, index := range indexes {
		createScript += `CREATE INDEX IF NOT EXISTS "` + runtimeTableName + `_` + index + `"
ON "` + runtimeTableName + `" USING gin (("object"->'` + index + `'));
`
	}

	for tenant := range dbs {
		if sharding {
			for _, shard := range Shards(tenant) {
				_, err = shard.Exec(createScript)
				if err != nil {
					return
				}
			}
		} else {
			_, err = Default(tenant).Exec(createScript)
			if err != nil {
				return
			}
		}
	}

	typeToTable[objType] = dbTable{
		ObjectType:       objType,
		ObjectTypeName:   objTypeName,
		RuntimeTableName: runtimeTableName,
		HistoryTableName: historyTableName,
		LockTableName:    lockTableName,
		Sharding:         sharding,
	}

	return
}

func NewObjectSet[tenantT ~string, objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string]() ObjectSet[tenantT, objT, idT, shardKeyT] {
	obj := new(objT)
	objType := reflect.TypeOf(*obj)
	return ObjectSet[tenantT, objT, idT, shardKeyT]{
		objType: objType,
	}
}

type ObjectSet[tenantT ~string, objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	objType reflect.Type
}

func (os ObjectSet[tenantT, objT, idT, shardKeyT]) Tenant(tenant tenantT) TenantObjectSet[objT, idT, shardKeyT] {
	return TenantObjectSet[objT, idT, shardKeyT]{
		objType: os.objType,
		tenant:  Tenant(tenant),
	}
}

type TenantObjectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	objType reflect.Type
	tenant  Tenant
}
