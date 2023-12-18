package db

import (
	"database/sql"
)

func (os ObjectSet[objT, idT, shardKeyT]) Delete(id idT, shardKeys ...shardKeyT) (err error) {

	table, ok := typeToTable[os.objType]
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
		dbs = dbsForShardKeys(shardKeys...)
	} else {
		dbs = []*sql.DB{Default()}
	}

	for _, db := range dbs {

		var bytes []byte

		err = db.
			QueryRow(`DELETE FROM "`+table.RuntimeTableName+`" WHERE id=$1`, id).
			Scan(&bytes)
		if err == sql.ErrNoRows {
			err = nil
			continue
		}
		if err != nil {
			return
		}
	}
	return
}
