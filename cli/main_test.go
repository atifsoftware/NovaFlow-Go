package main

import (
	"strings"
	"testing"
)

func TestParseMigrationSQL(t *testing.T) {
	sqlContent := `-- UP
CREATE TABLE users (id INT);
CREATE INDEX idx_users ON users(id);

-- DOWN
DROP TABLE users;
`

	up, down := parseMigrationSQL(sqlContent)

	if !strings.Contains(up, "CREATE TABLE users") {
		t.Errorf("expected UP block to contain CREATE TABLE, got: %q", up)
	}
	if strings.Contains(up, "DROP TABLE users") {
		t.Errorf("expected UP block NOT to contain DROP TABLE, got: %q", up)
	}
	if !strings.Contains(down, "DROP TABLE users") {
		t.Errorf("expected DOWN block to contain DROP TABLE, got: %q", down)
	}
	if strings.Contains(down, "CREATE TABLE users") {
		t.Errorf("expected DOWN block NOT to contain CREATE TABLE, got: %q", down)
	}
}

func TestParseMigrationSQL_NoDelimiters(t *testing.T) {
	sqlContent := "CREATE TABLE users (id INT);"
	up, down := parseMigrationSQL(sqlContent)

	if !strings.Contains(up, "CREATE TABLE users") {
		t.Errorf("expected UP block to contain CREATE TABLE, got: %q", up)
	}
	if strings.TrimSpace(down) != "" {
		t.Errorf("expected DOWN block to be empty, got: %q", down)
	}
}
