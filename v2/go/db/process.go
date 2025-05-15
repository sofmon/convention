package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Process(ctx convCtx.Context, where whereReady, process func(ctx convCtx.Context, obj objT) error, shardKeys ...shardKeyT) (count int, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	var dbs []*sql.DB
	dbs, err = dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	statement, params, err := where.statement()
	if err != nil {
		err = fmt.Errorf("error building where statement: %w", err)
		return
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+tos.table.RuntimeTableName+`" WHERE `+statement, params...)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		for rows.Next() {

			var (
				bytes []byte
				obj   objT
			)

			err = rows.Scan(&bytes)
			if err != nil {
				return
			}

			err = json.Unmarshal(bytes, &obj)
			if err != nil {
				return
			}

			err = process(ctx, obj)
			if err != nil {
				return
			}

			count++
		}

	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) ProcessWithMetadata(ctx convCtx.Context, where whereReady, process func(ctx convCtx.Context, obj ObjectWithMetadata[objT]) error, shardKeys ...shardKeyT) (count int, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	var dbs []*sql.DB
	dbs, err = dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	statement, params, err := where.statement()
	if err != nil {
		err = fmt.Errorf("error building where statement: %w", err)
		return
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object", "created_at", "created_by", "updated_at", "updated_by" FROM "`+tos.table.RuntimeTableName+`" WHERE `+statement, params...)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		for rows.Next() {

			var (
				bytes []byte
				obj   objT
				md    Metadata
			)

			err = rows.Scan(&bytes, &md.CreatedAt, &md.CreatedBy, &md.UpdatedAt, &md.UpdatedBy)
			if err != nil {
				return
			}

			err = json.Unmarshal(bytes, &obj)
			if err != nil {
				return
			}

			err = process(ctx, ObjectWithMetadata[objT]{obj, md})
			if err != nil {
				return
			}

			count++
		}

	}

	return
}
