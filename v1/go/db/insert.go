package db

import (
	"database/sql"
	"encoding/json"
	"errors"
)

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
