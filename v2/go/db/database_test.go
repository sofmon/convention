package db_test

import (
	"testing"

	convDB "github.com/sofmon/convention/v2/go/db"
)

func Test_open_and_close(t *testing.T) {

	err := convDB.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	dbs, err := convDB.DBs("messages", "test")
	if err != nil {
		t.Fatalf("DBs failed: %v", err)
	}

	if dbs == nil {
		t.Fatalf("DBs failed: nil")
	}

	if len(dbs) != 2 {
		t.Fatalf("DBs failed: %v", len(dbs))
	}

	err = convDB.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = convDB.Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

}
