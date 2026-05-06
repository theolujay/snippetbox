package models

import (
	"database/sql"
	"os"
	"testing"
)

func newTestDB(t *testing.T) *sql.DB {
	// Establish an sql.DB connection pool for the test database. Because the setup and teardown
	// scripts contain multiple SQL statements, use te "multiStatements=true" parameter in the
	// DSN; this instructs the MySQL database driver t support executing multiple SQL statements
	// in on db.Exec) call.
	db, err := sql.Open("mysql", "test_web:pass@/test_snippetbox?parseTime=true&multiStatements=true")
	if err != nil {
		t.Fatal(err)
	}

	// Read the setup SQL script from file and execute the statements
	script, err := os.ReadFile("./testdata/setup.sql")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(string(script))
	if err != nil {
		t.Fatal(err)
	}
	// Register a function which will automatically be called by Go
	// when the current test (or sub-test) which calls newTestDB()
	// has finished -- which the function will read and execute the
	// teardown script and close the database connection pool.
	t.Cleanup(func() {
		script, err := os.ReadFile("./testdata/teardown.sql")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.Exec(string(script))
		if err != nil {
			t.Fatal(err)
		}
		db.Close()
	})

	return db

}
