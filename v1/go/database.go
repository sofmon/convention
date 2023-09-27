package convention

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type DBEngine string

const (
	DBEnginePostgres DBEngine = "postgres"
)

type DBVersion string

type dbConnection struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type dbConnections struct {
	Default *dbConnection  `json:"default,omitempty"`
	Shards  []dbConnection `json:"shards,omitempty"`
}

type dbConfig struct {
	Engine   DBEngine                    `json:"engine"`
	Versions map[DBVersion]dbConnections `json:"versions"`
}

func (conn dbConnection) Open() (*sql.DB, error) {
	return sql.Open(
		"postgres",
		fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
			conn.Host, conn.Port, conn.Username, conn.Password, conn.Database,
		),
	)
}

func DBOpen(version DBVersion) (err error) {

	fullCfg, err := ConfigObject[dbConfig](configKeyDatabase)
	if err != nil {
		return
	}

	if fullCfg.Engine != DBEnginePostgres {
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

func DBClose() (err error) {
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

func DBDefault() *sql.DB {
	if defaultDB == nil {
		panic(ErrNoDBDefault)
	}
	return defaultDB
}

func DBShards() []*sql.DB {
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

type DBObject[idT, shardKeyT ~string] interface {
	ID() idT
	ShardKey() shardKeyT
	CreatedAt() time.Time
	CreatedBy() string
	UpdatedAt() time.Time
	UpdatedBy() string
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
		return DBShards()

	default:
		sksStr := make([]string, len(sks))
		for i, sk := range sks {
			sksStr[i] = string(sk)
		}
		return dbsByShardKeys(sksStr...)

	}

}

func DBRegisterObject[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](sharding bool) (err error) {

	objType := reflect.TypeOf(new(objT))
	objTypeName := objType.Name()

	runtimeTableName := toSnakeCase(objType.Name())
	historyTableName := runtimeTableName + historySuffix

	createScript := `CREATE TABLE IF NOT EXISTS "` + runtimeTableName + `" (
id text PRIMARY KEY,
created_at timestamp DEFAULT now(),
created_by text NOT NULL,
updated_at timestamp DEFAULT now(),
updated_by text NOT NULL,
object JSONB NULL
);
CREATE TABLE IF NOT EXISTS "` + historyTableName + `" (
id text NOT NULL,
created_at timestamp DEFAULT now(),
created_by text NOT NULL,
updated_at timestamp DEFAULT now(),
updated_by text NOT NULL,
object JSONB NULL
);`

	if sharding {
		for _, shard := range DBShards() {
			_, err = shard.Exec(createScript)
			if err != nil {
				return
			}
		}
	} else {
		_, err = DBDefault().Exec(createScript)
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

func DBInsertObject[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](obj objT) (err error) {

	objType := reflect.TypeOf(obj)

	table, ok := typeToTable[objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(obj.ShardKey()))
	} else {
		db = DBDefault()
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
		obj.ID(), obj.CreatedAt(), obj.CreatedBy(), obj.UpdatedAt(), obj.UpdatedBy(), bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		obj.ID())
	if err != nil {
		return
	}

	return
}

func DBUpdateObject[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](obj objT) (err error) {

	objType := reflect.TypeOf(obj)

	table, ok := typeToTable[objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(obj.ShardKey()))
	} else {
		db = DBDefault()
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
		bytes, obj.UpdatedAt(), obj.UpdatedBy(), obj.ID())
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		obj.ID())
	if err != nil {
		return
	}

	return
}

func DBUpsertObject[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](obj objT) (err error) {

	objType := reflect.TypeOf(obj)

	table, ok := typeToTable[objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(obj.ShardKey()))
	} else {
		db = DBDefault()
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
		obj.ID(), obj.CreatedAt(), obj.CreatedBy(), obj.UpdatedAt(), obj.UpdatedBy(), bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		obj.ID())
	if err != nil {
		return
	}

	return
}

func DBSelectObject[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](id idT, shardKeys ...shardKeyT) (obj objT, err error) {

	objType := reflect.TypeOf(obj)

	table, ok := typeToTable[objType]
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
		dbs = []*sql.DB{DBDefault()}
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

func DBSelectObjects[objT DBObject[idT, shardKeyT], idT ~string, shardKeyT ~string](where string, params ...any) (obs []objT, err error) {

	objType := reflect.TypeOf(new(objT))

	table, ok := typeToTable[objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = DBShards()
	} else {
		dbs = []*sql.DB{DBDefault()}
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
