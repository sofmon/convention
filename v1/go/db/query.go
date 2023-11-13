package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
)

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

func (os ObjectSet[objT, idT, shardKeyT]) SelectByID(id idT, shardKeys ...shardKeyT) (obj *objT, err error) {

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
			QueryRow(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		obj = new(objT)
		err = json.Unmarshal(bytes, obj)
		if err != nil {
			return
		}

	}
	return
}

func (os ObjectSet[objT, idT, shardKeyT]) Delete(id idT, shardKeys ...shardKeyT) (err error) {

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
			QueryRow(`DELETE FROM "`+table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}
	}
	return
}

type where struct {
	statement string
	params    []any
}

func Where(statement string, params ...any) where {
	return where{
		statement: statement,
		params:    params,
	}
}

func (os ObjectSet[objT, idT, shardKeyT]) Select(where where, shardKeys ...shardKeyT) (obs []objT, err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = dbsForShardKeys(shardKeys...)
	} else {
		dbs = []*sql.DB{Default()}
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE `+where.statement, where.params...)
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

type exec struct {
	statement string
	params    []any
}

func Exec(statement string, params ...any) exec {
	return exec{
		statement: statement,
		params:    params,
	}
}

func (os ObjectSet[objT, idT, shardKeyT]) Exec(exec exec, shardKeys ...shardKeyT) (err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = dbsForShardKeys(shardKeys...)
	} else {
		dbs = []*sql.DB{Default()}
	}

	txs := make([]*sql.Tx, len(dbs))
	for i, db := range dbs {
		txs[i], err = db.Begin()
		if err != nil {
			return
		}
		defer func() {
			if err != nil {
				err = errors.Join(
					err,
					txs[i].Rollback(),
				)
				return
			}
			err = txs[i].Commit()
		}()
	}

	for _, tx := range txs {
		_, err = tx.Exec(exec.statement, exec.params...)
		if err != nil {
			return
		}
	}

	return
}
