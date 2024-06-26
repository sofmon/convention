package db

import (
	"database/sql"
	"errors"
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) Exec(exec exec, shardKeys ...shardKeyT) (err error) {

	table, ok := typeToTable[tos.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = dbsForShardKeys(tos.tenant, shardKeys...)
	} else {
		dbs = []*sql.DB{Default(tos.tenant)}
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
