package db

import (
	"database/sql"
	"errors"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type exec struct {
	statement string
	params    []any
}

func Exec(statement string, params ...any) exec {
	return exec{
		statement: statement,
		params:    params,
	}
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Exec(ctx convCtx.Context, exec exec, shardKeys ...shardKeyT) (err error) {

	if !tos.isInitialized() {
		err = ErrObjectTypeNotRegistered
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
		for _, tx := range txs {
			if err != nil {
				err = errors.Join(
					err,
					tx.Rollback(),
				)
				return
			}
			err = tx.Commit()
		}
	}()

	for _, tx := range txs {
		_, err = tx.Exec(exec.statement, exec.params...)
		if err != nil {
			return
		}
	}

	return
}
