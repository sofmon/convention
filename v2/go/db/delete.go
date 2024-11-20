package db

import (
	"database/sql"
	"errors"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Delete(ctx convCtx.Context, id idT, shardKeys ...shardKeyT) (err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	var dbs []*sql.DB
	dbs, err = dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	txs := make([]*sql.Tx, len(dbs))
	for i, db := range dbs {
		txs[i], err = db.Begin()
		if err != nil {
			return
		}
	}
	defer func() {
		if err != nil {
			for _, tx := range txs {
				err = errors.Join(
					err,
					tx.Rollback(),
				)
			}
			return
		} else {
			for _, tx := range txs {
				err = errors.Join(
					err,
					tx.Commit(),
				)
			}
		}
	}()

	now, user := ctx.Now(), ctx.User()

	for _, tx := range txs {

		res, e := tx.Exec(`DELETE FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id)
		if e != nil {
			err = errors.Join(err, e)
			break
		}

		ra, e := res.RowsAffected()
		if e != nil {
			err = errors.Join(err, e)
			break
		}

		if ra == 0 {
			continue // no record was deleted, move on
		}

		// ensure history record is created indicating deletion
		_, e = tx.Exec(`INSERT INTO "`+tos.table.HistoryTableName+`"
		("id","created_at","created_by","updated_at","updated_by","object")
		VALUES($1,$2,$3,$4,$5,NULL)`,
			id, now, user, now, user)
		if e != nil {
			err = errors.Join(err, e)
			break
		}
	}
	if err != nil {
		return
	}

	return
}
