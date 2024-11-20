package db

import (
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type Lock[objT Object[idT, shardKeyT], idT, shardKeyT ~string] struct {
	tos TenantObjectSet[objT, idT, shardKeyT]
	sk  shardKeyT
	id  idT
}

func (l Lock[objT, idT, shardKeyT]) Unlock() (err error) {

	db, err := dbByShardKey(l.tos.vault, l.tos.tenant, string(l.sk))
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

	if !tos.isInitialized() {
		err = ErrObjectTypeNotRegistered
		return
	}

	key := obj.DBKey()

	db, err := dbByShardKey(tos.vault, tos.tenant, string(key.ShardKey))
	if err != nil {
		return
	}

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
		sk:  key.ShardKey,
		id:  key.ID,
	}

	return
}
