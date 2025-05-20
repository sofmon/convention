package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Update(ctx convCtx.Context, obj objT) (err error) {

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
	if err == sql.ErrNoRows {
		err = fmt.Errorf("object with ID '%s' does not exist", key.ID)
		return
	}
	if err != nil {
		return
	}

	md.UpdatedAt = ctx.Now()
	md.UpdatedBy = ctx.User()

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

	_, err = tx.Exec(`UPDATE "`+tos.table.RuntimeTableName+`" SET "object"=$1, "updated_at"=$2, "updated_by"=$3 WHERE "id"=$4`,
		bytes, md.UpdatedAt, md.UpdatedBy, key.ID)
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) SafeUpdate(ctx convCtx.Context, from, to objT) (err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	fromKey, toKey := from.DBKey(), to.DBKey()

	if fromKey.ID != toKey.ID {
		err = errors.New("cannot safely update object with different IDs")
		return
	}

	if fromKey.ShardKey != toKey.ShardKey {
		err = errors.New("cannot safely update object with different shard keys")
		return
	}

	db, err := dbByShardKey(tos.vault, tos.tenant, string(fromKey.ShardKey))
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

	var (
		cmp     objT
		cmpHash string
		md      Metadata
	)
	row := tx.QueryRow(`SELECT "object", md5("object"), "created_at", "created_by", "updated_at", "updated_by"  FROM "`+tos.table.RuntimeTableName+`" WHERE "id"=$1 FOR UPDATE NOWAIT`, fromKey.ID)
	err = row.Scan(&cmp, &cmpHash, &md.CreatedAt, &md.CreatedBy, &md.UpdatedAt, &md.UpdatedBy)
	if err == sql.ErrNoRows {
		return fmt.Errorf("object with ID '%s' does not exist", fromKey.ID)
	}
	if err != nil {
		return
	}

	md.UpdatedAt = ctx.Now()
	md.UpdatedBy = ctx.User()

	cmpBytes, err := json.Marshal(cmp)
	if err != nil {
		return
	}

	fromBytes, err := json.Marshal(from)
	if err != nil {
		return
	}

	if string(cmpBytes) != string(fromBytes) {
		return fmt.Errorf("object with ID '%s' has been modified since it was retrieved", fromKey.ID)
	}

	for _, compute := range tos.compute {
		err = compute(ctx, md, &to)
		if err != nil {
			return
		}
	}

	toBytes, err := json.Marshal(to)
	if err != nil {
		return
	}

	res, err := tx.Exec(`UPDATE "`+tos.table.RuntimeTableName+`" SET "object"=$1, "updated_at"=$2, "updated_by"=$3 WHERE "id"=$4 AND md5("object")=$5`,
		toBytes, md.UpdatedAt, md.UpdatedBy, toKey.ID, cmpHash)
	if err != nil {
		return
	}

	count, err := res.RowsAffected()
	if err != nil {
		return
	}

	if count == 0 {
		return fmt.Errorf("object with ID '%s' has been modified since it was retrieved", fromKey.ID)
	}

	_, err = tx.Exec(`INSERT INTO "`+tos.table.HistoryTableName+`" SELECT "id", "created_at", "created_by", "updated_at", "updated_by", "object" FROM "`+tos.table.RuntimeTableName+`" WHERE "id"=$1`,
		toKey.ID)
	if err != nil {
		return
	}

	return
}
