package db

import (
	"database/sql"
	"encoding/json"
	"errors"

	convCtx "github.com/sofmon/convention/lib/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Insert(ctx convCtx.Context, obj objT) (err error) {

	err = tos.prepare()
	if err != nil {
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

	var md Metadata
	md.CreatedAt = ctx.Now()
	md.CreatedBy = ctx.User()
	md.UpdatedAt = md.CreatedAt
	md.UpdatedBy = md.CreatedBy

	for _, compute := range tos.compute {
		err = compute(ctx, md, &obj)
		if err != nil {
			return
		}
	}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)`,
		key.ID, md.CreatedAt, md.CreatedBy, md.UpdatedAt, md.UpdatedBy, bytes)
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

	err = tos.prepare()
	if err != nil {
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

	var md Metadata

	err = tx.QueryRow(`SELECT "created_at", "created_by", "updated_at", "updated_by" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, key.ID).
		Scan(&md.CreatedAt, &md.CreatedBy, &md.UpdatedAt, &md.UpdatedBy)
	if err != nil && err != sql.ErrNoRows {
		return
	}
	if err == sql.ErrNoRows {
		md.CreatedAt = ctx.Now()
		md.CreatedBy = ctx.User()
		md.UpdatedAt = md.CreatedAt
		md.UpdatedBy = md.CreatedBy
	} else {
		md.UpdatedAt = ctx.Now()
		md.UpdatedBy = ctx.User()
	}

	for _, compute := range tos.compute {
		err = compute(ctx, md, &obj)
		if err != nil {
			return
		}
	}

	bytes, err := json.Marshal(obj)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT ("id")
DO UPDATE SET "updated_at"=$4,"updated_by"=$5,"object"=$6`,
		key.ID, md.CreatedAt, md.CreatedBy, md.UpdatedAt, md.UpdatedBy, bytes)
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) UpsertWithMetadata(ctx convCtx.Context, obj ObjectWithMetadata[objT]) (err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	key := obj.Object.DBKey()

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

	for _, compute := range tos.compute {
		err = compute(ctx, obj.Metadata, &obj.Object)
		if err != nil {
			return
		}
	}

	bytes, err := json.Marshal(obj.Object)
	if err != nil {
		return
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.RuntimeTableName+`"
("id","created_at","created_by","updated_at","updated_by","object")
VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT ("id")
DO UPDATE SET "updated_at"=$4,"updated_by"=$5,"object"=$6`,
		key.ID, obj.Metadata.CreatedAt, obj.Metadata.CreatedBy, obj.Metadata.UpdatedAt, obj.Metadata.UpdatedBy, bytes)
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
