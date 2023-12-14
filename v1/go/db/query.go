package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func (os ObjectSet[objT, idT, shardKeyT]) SafeUpdate(from, to objT) (err error) {

	table, ok := typeToTable[os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	fromTrail, toTrail := from.Trail(), to.Trail()

	if fromTrail.ID != toTrail.ID {
		err = errors.New("cannot safely update object with different IDs")
		return
	}

	if fromTrail.ShardKey != toTrail.ShardKey {
		err = errors.New("cannot safely update object with different shard keys")
		return
	}

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(fromTrail.ShardKey))
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

	var (
		cmp     objT
		cmpHash string
	)
	row := tx.QueryRow(`SELECT "object", md5("object") FROM "`+table.RuntimeTableName+`" WHERE "id"=$1 FOR UPDATE NOWAIT`, fromTrail.ID)
	err = row.Scan(&cmp, &cmpHash)
	if err == sql.ErrNoRows {
		return fmt.Errorf("object with ID '%s' does not exist", fromTrail.ID)
	}
	if err != nil {
		return
	}

	cmpBytes, err := json.Marshal(cmp)
	if err != nil {
		return
	}

	fromBytes, err := json.Marshal(from)
	if err != nil {
		return
	}

	if string(cmpBytes) != string(fromBytes) {
		return fmt.Errorf("object with ID '%s' has been modified since it was retrieved", fromTrail.ID)
	}

	toBytes, err := json.Marshal(to)
	if err != nil {
		return
	}

	res, err := tx.Exec(`UPDATE "`+table.RuntimeTableName+`" SET "object"=$1, "updated_at"=$2, "updated_by"=$3 WHERE "id"=$4 AND md5("object")=$5`,
		toBytes, toTrail.UpdatedAt, toTrail.UpdatedBy, toTrail.ID, cmpHash)
	if err != nil {
		return
	}

	count, err := res.RowsAffected()
	if err != nil {
		return
	}

	if count == 0 {
		return fmt.Errorf("object with ID '%s' has been modified since it was retrieved", fromTrail.ID)
	}

	_, err = tx.Exec(`INSERT INTO "`+table.HistoryTableName+`" SELECT * FROM "`+table.RuntimeTableName+`" WHERE "id"=$1`,
		toTrail.ID)
	if err != nil {
		return
	}

	return
}
