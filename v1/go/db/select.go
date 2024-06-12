package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectAll() (obs []objT, err error) {

	table, ok := typeToTable[tos.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = Shards(tos.tenant)
	} else {
		dbs = []*sql.DB{Default(tos.tenant)}
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "` + table.RuntimeTableName + `"`)
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) SelectByID(id idT, shardKeys ...shardKeyT) (obj *objT, err error) {

	table, ok := typeToTable[tos.objType]
	if !ok {
		err = ErrObjectTypeNotRegistered
		return
	}

	if !table.Sharding && len(shardKeys) > 0 {
		err = ErrObjectNotUsingShards
		return
	}

	var dbs []*sql.DB
	if table.Sharding {
		dbs = dbsForShardKeys(tos.tenant, shardKeys...)
	} else {
		dbs = []*sql.DB{Default(tos.tenant)}
	}

	for _, db := range dbs {

		var bytes []byte

		err = db.
			QueryRow(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE id=$1`, id).
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

type where struct {
	statement string
	params    []any
}

func Where(statement string, params ...any) where {
	return where{
		statement: statement,
		params:    params,
	}
}

func (w where) Limit(limit int) where {
	w.statement += fmt.Sprintf(` LIMIT %d`, limit)
	return w
}

func (tos TenantObjectSet[objT, idT, shardKeyT]) Select(where where, shardKeys ...shardKeyT) (obs []objT, err error) {

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

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE `+where.statement, where.params...)
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
