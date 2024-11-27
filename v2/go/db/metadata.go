package db

import (
	"database/sql"
	"time"

	convAuth "github.com/sofmon/convention/v2/go/auth"
	convCtx "github.com/sofmon/convention/v2/go/ctx"
)

type Metadata struct {
	CreatedAt time.Time
	CreatedBy convAuth.User
	UpdatedAt time.Time
	UpdatedBy convAuth.User
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Metadata(ctx convCtx.Context, id idT, shardKeys ...shardKeyT) (res *Metadata, err error) {

	err = tos.prepare()
	if err != nil {
		return
	}

	dbs, err := dbsForShardKeys(tos.vault, tos.tenant, shardKeys...)
	if err != nil {
		return
	}

	var md Metadata

	for _, db := range dbs {

		err = db.
			QueryRow(`SELECT "created_at","created_by","updated_at","updated_by" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&md.CreatedAt, &md.CreatedBy, &md.UpdatedAt, &md.UpdatedBy)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		res = &md
		return
	}
	return
}
