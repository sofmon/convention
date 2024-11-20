package db

import (
	"encoding/json"
	"errors"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Insert(ctx convCtx.Context, obj objT) (err error) {

	if !tos.isInitialized() {
		err = ErrObjectTypeNotRegistered
		return
	}

	key := obj.DBKey()

	db, err := dbByShardKey(tos.vault, tos.tenant, string(key.ShardKey))
	if err != nil {
		return
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

	now, user := ctx.Now(), ctx.User()

	_, err = tx.Exec(`INSERT INTO "`+tos.table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)`,
		key.ID, now, user, now, user, bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.HistoryTableName+`" SELECT "id", "created_at", "created_by", "updated_at", "updated_by", "object" FROM "`+tos.table.RuntimeTableName+`" WHERE "id"=$1`,
		key.ID)
	if err != nil {
		return
	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Upsert(ctx convCtx.Context, obj objT) (err error) {

	if !tos.isInitialized() {
		err = ErrObjectTypeNotRegistered
		return
	}

	key := obj.DBKey()

	db, err := dbByShardKey(tos.vault, tos.tenant, string(key.ShardKey))
	if err != nil {
		return
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

	now, user := ctx.Now(), ctx.User()

	_, err = tx.Exec(`INSERT INTO "`+tos.table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT ("id")
DO UPDATE SET "updated_at"=$4,"updated_by"=$5,"object"=$6`,
		key.ID, now, user, now, user, bytes)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.HistoryTableName+`" SELECT "id", "created_at", "created_by", "updated_at", "updated_by", "object" FROM "`+tos.table.RuntimeTableName+`" WHERE "id"=$1`,
		key.ID)
	if err != nil {
		return
	}

	return
}
