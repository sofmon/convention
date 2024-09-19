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

func (tos TenantObjectSet[objT, idT, shardKeyT]) Select(where *where, shardKeys ...shardKeyT) (obs []objT, err error) {

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

	statement, params, err := where.statement()
	if err != nil {
		err = fmt.Errorf("error building where statement: %w", err)
		return
	}

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE `+statement, params...)
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
