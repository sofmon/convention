package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectAll(ctx convCtx.Context) (obs []objT, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := DBs(tos.vault, tos.tenant)
	if err != nil {
		return
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "` + tos.table.RuntimeTableName + `"`)
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

			obs = append(obs, obj)
		}

	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectAllWithMetadata(ctx convCtx.Context) (obs ListWithMetadata[objT], err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := DBs(tos.vault, tos.tenant)
	if err != nil {
		return
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object", "created_at", "created_by", "updated_at", "updated_by" FROM "` + tos.table.RuntimeTableName + `"`)
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

			obs = append(obs, ObjectWithMetadata[objT]{obj, md})
		}

	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectByID(ctx convCtx.Context, id idT, shardKeys ...shardKeyT) (obj *objT, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	for _, db := range dbs {

		var bytes []byte

		err = db.
			QueryRow(`SELECT "object" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = nil
			continue
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectByIDWithMetadata(ctx convCtx.Context, id idT, shardKeys ...shardKeyT) (obj *ObjectWithMetadata[objT], err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	for _, db := range dbs {

		var (
			bytes []byte
			o     objT
			md    Metadata
		)

		err = db.
			QueryRow(`SELECT "object", "created_at", "created_by", "updated_at", "updated_by" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes, &md.CreatedAt, &md.CreatedBy, &md.UpdatedAt, &md.UpdatedBy)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		err = json.Unmarshal(bytes, &o)
		if err != nil {
			return
		}

		obj = &ObjectWithMetadata[objT]{o, md}
	}
	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Select(ctx convCtx.Context, where whereReady, shardKeys ...shardKeyT) (obs []objT, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
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

			obs = append(obs, obj)
		}

	}

	return
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectWithMetadata(ctx convCtx.Context, where whereReady, shardKeys ...shardKeyT) (obs ListWithMetadata[objT], err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
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

			obs = append(obs, ObjectWithMetadata[objT]{obj, md})
		}

	}

	return
}
