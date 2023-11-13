package db

import (
	"database/sql"
	"time"
)

type Lock[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	os ObjectSet[objT, idT, shardKeyT]
	sk shardKeyT
	id idT
}

func (l Lock[objT, idT, shardKeyT]) Unlock() (err error) {
	table, ok := typeToTable[l.os.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var db *sql.DB
	if table.Sharding {
		db = dbByShardKey(string(l.sk))
	} else {
		db = Default()
	}

	_, err = db.Exec(`DELETE FROM "`+table.LockTableName+`"WHERE "id"=$1;`, l.id)
	if err != nil {
		return
	}

	return
}

func (os ObjectSet[objT, idT, shardKeyT]) Lock(obj objT, desc string) (lock *Lock[objT, idT, shardKeyT], err error) {

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

	res, err := db.Exec(`INSERT INTO "`+table.LockTableName+`"
("id","created_at","description")
VALUES($1,$2,$3)
ON CONFLICT ("id") DO NOTHING;`,
		trail.ID, time.Now().UTC(), desc)
	if err != nil {
		return
	}

	count, err := res.RowsAffected()
	if err != nil {
		return
	}

	if count == 0 {
		return // someone was faster, no lock
	}

	lock = &Lock[objT, idT, shardKeyT]{
		os: os,
		sk: trail.ShardKey,
		id: trail.ID,
	}

	return
}
