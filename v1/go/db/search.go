package db

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

func (tos TenantObjectSet[objT, idT, shardKeyT]) Search(text string, shardKeys ...shardKeyT) (obs []objT, err error) {

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
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE "text_search" @@ to_tsquery('english', $1);`, toTSQuery(text))
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

func (tos TenantObjectSet[objT, idT, shardKeyT]) SearchWhere(text string, where where, shardKeys ...shardKeyT) (obs []objT, err error) {

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

	params := append(where.params, toTSQuery(text))

	for _, db := range dbs {

		var rows *sql.Rows
		rows, err = db.Query(`SELECT "object" FROM "`+table.RuntimeTableName+`" WHERE (`+where.statement+`) AND "text_search" @@ to_tsquery('english', $`+strconv.Itoa(len(params)+1)+`)`, params...)
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

func toTSQuery(input string) string {

	// Step 1: Remove non-alphanumeric characters (except spaces)
	re := regexp.MustCompile(`[^\w\s]`)
	cleaned := re.ReplaceAllString(input, "")

	// Step 2: Replace multiple spaces with a single space
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")

	// Step 3: Trim leading and trailing spaces (if any)
	cleaned = strings.TrimSpace(cleaned)

	// Step 4: Replace spaces with the '&' operator
	formatted := strings.ReplaceAll(cleaned, " ", " & ")

	return formatted
}
