package db

import (
	"database/sql"
	"time"

	convAuth "github.com/sofmon/convention/lib/auth"
	convCtx "github.com/sofmon/convention/lib/ctx"
)

type Metadata struct {
	CreatedAt time.Time     `json:"created_at"`
	CreatedBy convAuth.User `json:"created_by"`
	UpdatedAt time.Time     `json:"updated_at"`
	UpdatedBy convAuth.User `json:"updated_by"`
}

type ObjectWithMetadata[T any] struct {
	Object   T        `json:"object"`
	Metadata Metadata `json:"metadata"`
}

type ListWithMetadata[T any] []ObjectWithMetadata[T]

func (lwm ListWithMetadata[T]) Objects() []T {
	objects := make([]T, len(lwm))
	for i, lwm := range lwm {
		objects[i] = lwm.Object
	}
	return objects
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

	res = &Metadata{}

	for _, db := range dbs {

		err = db.
			QueryRow(`SELECT "created_at","created_by","updated_at","updated_by" FROM "`+tos.table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&res.CreatedAt, &res.CreatedBy, &res.UpdatedAt, &res.UpdatedBy)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}

		return
	}
	return
}
