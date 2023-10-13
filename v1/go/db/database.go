package db

import (
	_ "github.com/lib/pq"

	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"reflect"
	"regexp"
	"strings"
	"time"

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
	Sharding         bool
}

const (
	historySuffix = "_history"
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

func dbsForShardKeys[shardKeyT ~string](sks ...shardKeyT) []*sql.DB {

	switch len(sks) {

	case 1:
		return []*sql.DB{dbByShardKey(string(sks[0]))}

	case 0:
		return Shards()

	default:
		sksStr := make([]string, len(sks))
		for i, sk := range sks {
			sksStr[i] = string(sk)
		}
		return dbsByShardKeys(sksStr...)

	}

}

func RegisterObject[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string](sharding bool) (err error) {

	obj := new(objT)
	objType := reflect.TypeOf(*obj)
	objTypeName := objType.Name()

	runtimeTableName := toSnakeCase(objType.Name())
	historyTableName := runtimeTableName + historySuffix

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
);`

	if sharding {
		for _, shard := range Shards() {
			_, err = shard.Exec(createScript)
			if err != nil {
				return
			}
		}
	} else {
		_, err = Default().Exec(createScript)
		if err != nil {
			return
		}
	}

	typeToTable[objType] = dbTable{
		ObjectType:       objType,
		ObjectTypeName:   objTypeName,
		RuntimeTableName: runtimeTableName,
		HistoryTableName: historyTableName,
		Sharding:         sharding,
	}

	return
}

func NewObjectSet[objT Object[idT, shardKeyT], idT ~string, shardKeyT ~string]() ObjectSet[objT, idT, shardKeyT] {
	obj := new(objT)
	objType := reflect.TypeOf(*obj)
	return ObjectSet[objT, idT, shardKeyT]{
		objType: objType,
	}
}

type ObjectSet[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	objType reflect.Type
}

func (os ObjectSet[objT, idT, shardKeyT]) Insert(obj objT) (err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	trail := obj.Trail()

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(trail.ShardKey))
	} else {
		db = Default()
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			err = errors.Join(
				err,
				tx.Rollback(),
			)
			return
		}
		err = tx.Commit()
	}()

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)`,
		trail.ID, trail.CreatedAt, trail.CreatedBy, trail.UpdatedAt, trail.UpdatedBy, bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		trail.ID)
	if err != nil {
		return
	}

	return
}

func (os ObjectSet[objT, idT, shardKeyT]) Update(obj objT) (err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	trail := obj.Trail()

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(trail.ShardKey))
	} else {
		db = Default()
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			err = errors.Join(
				err,
				tx.Rollback(),
			)
			return
		}
		err = tx.Commit()
	}()

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	_, err = tx.Exec(`UPDATE "`+table.RuntimeTableName+`" SET "object"=$1, "updated_at"=$2, "updated_by"=$3 WHERE "id"=$4`,
		bytes, trail.UpdatedAt, trail.UpdatedBy, trail.ID)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		trail.ID)
	if err != nil {
		return
	}

	return
}

func (os ObjectSet[objT, idT, shardKeyT]) Upsert(obj objT) (err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	trail := obj.Trail()

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(trail.ShardKey))
	} else {
		db = Default()
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			err = errors.Join(
				err,
				tx.Rollback(),
			)
			return
		}
		err = tx.Commit()
	}()

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT ("id")
DO UPDATE SET "updated_at"=$4,"updated_by"=$5,"object"=$6`,
		trail.ID, trail.CreatedAt, trail.CreatedBy, trail.UpdatedAt, trail.UpdatedBy, bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		trail.ID)
	if err != nil {
		return
	}

	return
}

func (os ObjectSet[objT, idT, shardKeyT]) SelectByID(id idT, shardKeys ...shardKeyT) (obj objT, err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	if !table.Sharding && len(shardKeys) > 0 {
		err = ErrObjectNotUsingShards
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = dbsForShardKeys(shardKeys...)
	} else {
		dbs = []*sql.DB{Default()}
	}

	for _, db := range dbs {

		var bytes []byte

		err = db.
			QueryRow(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE id = $1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		err = json.Unmarshal(bytes, &obj)
		if err != nil {
			return
		}

	}
	return
}

func (os ObjectSet[objT, idT, shardKeyT]) Select(where string, params ...any) (obs []objT, err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = Shards()
	} else {
		dbs = []*sql.DB{Default()}
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE `+where, params...)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		for rows.Next() {

			var (
				bytes []byte
				obj   objT
			)

			err = rows.Scan(&bytes)
			if err != nil {
				return
			}

			err = json.Unmarshal(bytes, &obj)
			if err != nil {
				return
			}

			obs = append(obs, obj)
		}

	}

	return
}
