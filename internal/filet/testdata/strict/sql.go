package fixture

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
)

const table = "users"

func concatenated(db *sql.DB, name string) error {
	_, err := db.Exec("DELETE FROM users WHERE name = '" + name + "'") // want
	return err
}

func sprintfed(ctx context.Context, db *sql.DB, column string) (*sql.Rows, error) {
	return db.QueryContext(ctx, fmt.Sprintf("SELECT %s FROM users", column)) // want
}

func builtInSteps(db *sql.DB, filter string) (*sql.Rows, error) {
	q := "SELECT id FROM users"
	if filter != "" {
		q += " WHERE name = '" + filter + "'"
	}
	return db.Query(q) // want
}

func quotedWithStrconv(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query("SELECT id FROM " + strconv.Quote(name)) // want
}

func placeholders(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query("SELECT id FROM users WHERE name = $1", name)
}

func constantParts(db *sql.DB) (*sql.Rows, error) {
	return db.Query("SELECT id FROM " + table)
}

func numericParts(db *sql.DB, limit int) (*sql.Rows, error) {
	q := fmt.Sprintf("SELECT id FROM users LIMIT %d", limit)
	return db.Query(q + " OFFSET " + strconv.Itoa(limit))
}

func safeSteps(db *sql.DB, name string) (*sql.Rows, error) {
	q := "SELECT id FROM users"
	if name != "" {
		q += " WHERE name = $1"
	}
	return db.Query(q, name)
}

func quotedIdentifier(db *sql.DB, name string) (*sql.Rows, error) {
	return db.Query("SELECT id FROM " + quoteIdentifier(name))
}

func passedThrough(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	return db.Query(query, args...)
}

func quoteIdentifier(s string) string { return `"` + s + `"` }
