package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type Lock[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	tos TenantObjectSet[objT, idT, shardKeyT]
	si  int
	id  idT
}

func (l Lock[objT, idT, shardKeyT]) Unlock() (err error) {

	db, err := dbByIndex(l.tos.vault, l.tos.tenant, l.si)
	if err != nil {
		return
	}

	_, err = db.Exec(`DELETE FROM "`+l.tos.table.LockTableName+`"WHERE "id"=$1;`, l.id)
	if err != nil {
		return
	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Lock(ctx convCtx.Context, obj objT, desc string) (lock *Lock[objT, idT, shardKeyT], err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	key := obj.DBKey()

	dbs, err := DBs(tos.vault, tos.tenant)
	if err != nil {
		return
	}

	si := indexByShardKey(string(key.ShardKey), len(dbs))

	db := dbs[si]

	res, err := db.Exec(`INSERT INTO "`+tos.table.LockTableName+`"
("id","created_at","description")
VALUES($1,$2,$3)
ON CONFLICT ("id") DO NOTHING;`,
		key.ID, ctx.Now(), desc)
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
		tos: tos,
		si:  si,
		id:  key.ID,
	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectByIDAndLock(ctx convCtx.Context, id idT, desc string, shardKeys ...shardKeyT) (obj *objT, lock *Lock[objT, idT, shardKeyT], err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := DBs(tos.vault, tos.tenant)
	if err != nil {
		return
	}

	sis := make(map[int]any)
	if len(shardKeys) == 0 {
		for i := range dbs {
			sis[i] = nil
		}
	} else {
		for _, key := range shardKeys {
			i := indexByShardKey(string(key), len(dbs))
			sis[i] = nil
		}
	}

	for si, db := range dbs {

		if _, ok := sis[si]; !ok {
			continue
		}

		var exists bool
		err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM "`+tos.table.RuntimeTableName+`" WHERE id = $1)`, id).
			Scan(&exists)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}
		if !exists {
			continue
		}

		var execRes sql.Result
		execRes, err = db.Exec(`INSERT INTO "`+tos.table.LockTableName+`"
("id","created_at","description")
VALUES($1,$2,$3)
ON CONFLICT ("id") DO NOTHING;`,
			id, ctx.Now(), desc)
		if err != nil {
			return
		}

		var count int64
		count, err = execRes.RowsAffected()
		if err != nil {
			return
		}

		if count == 0 {
			return // someone was faster, no lock, no object
		}

		lock = &Lock[objT, idT, shardKeyT]{
			tos: tos,
			si:  si,
			id:  id,
		}

		var bytes []byte

		err = db.
			QueryRow(`SELECT "object" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = fmt.Errorf("object not found, even though lock was acquired")
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
