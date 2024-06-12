package db

import (
	"database/sql"
	"encoding/json"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Process(where where, process func(obj objT) error, shardKeys ...shardKeyT) (count int, err error) {

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

			err = process(obj)
			if err != nil {
				return
			}

			count++
		}

	}

	return
}
