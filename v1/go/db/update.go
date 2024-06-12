package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Update(obj objT) (err error) {

	table, ok := typeToTable[tos.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	trail := obj.Trail()

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(tos.tenant, string(trail.ShardKey))
	} else {
		db = Default(tos.tenant)
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) SafeUpdate(from, to objT) (err error) {

	table, ok := typeToTable[tos.objType]
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
		db = dbByShardKey(tos.tenant, string(fromTrail.ShardKey))
	} else {
		db = Default(tos.tenant)
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
