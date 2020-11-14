package gomigrate

import (
	"strconv"
	"strings"
)

// Migratable is migration interface
type Migratable interface {
	SelectMigrationTableSQL() string
	CreateMigrationTableSQL() string
	GetMigrationSQL() string
	MigrationLogInsertSQL() string
	MigrationLogDeleteSQL() string
	GetMigrationCommands(string) []string
}

// Postgres migrator
type Postgres struct{}

// SelectMigrationTableSQL gets table names from db catalog
func (p Postgres) SelectMigrationTableSQL() string {
	return "SELECT tablename FROM pg_catalog.pg_tables WHERE tablename = $1"
}

// CreateMigrationTableSQL gets SQL to create table for holding migrations
func (p Postgres) CreateMigrationTableSQL() string {
	return `CREATE TABLE gomigrate (
                  id           SERIAL       PRIMARY KEY,
                  migration_id BIGINT       UNIQUE NOT NULL
                )`
}

// GetMigrationSQL gets migration SQL for given id
func (p Postgres) GetMigrationSQL() string {
	return `SELECT migration_id FROM gomigrate WHERE migration_id = $1`
}

// MigrationLogInsertSQL gets insert SQL for migration
func (p Postgres) MigrationLogInsertSQL() string {
	return "INSERT INTO gomigrate (migration_id) values ($1)"
}

// MigrationLogDeleteSQL returns SQL for deleting a migration"
func (p Postgres) MigrationLogDeleteSQL() string {
	return "DELETE FROM gomigrate WHERE migration_id = $1"
}

// GetMigrationCommands return SQL commands
func (p Postgres) GetMigrationCommands(SQL string) []string {
	return []string{SQL}
}

// CockroachDB migrator
type CockroachDB struct {
	Postgres
}

// MySQL adapter
type MySQL struct{}

// SelectMigrationTableSQL gets table names from db catalog
func (m MySQL) SelectMigrationTableSQL() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_name = ? AND table_schema = (SELECT DATABASE())"
}

// CreateMigrationTableSQL gets insert SQL for migration
func (m MySQL) CreateMigrationTableSQL() string {
	return `CREATE TABLE gomigrate (
                  id           INT          NOT NULL AUTO_INCREMENT,
                  migration_id BIGINT       NOT NULL UNIQUE,
                  PRIMARY KEY (id)
                )`
}

// GetMigrationSQL gets migration SQL for given id
func (m MySQL) GetMigrationSQL() string {
	return `SELECT migration_id FROM gomigrate WHERE migration_id = ?`
}

// MigrationLogInsertSQL gets insert SQL for migration
func (m MySQL) MigrationLogInsertSQL() string {
	return "INSERT INTO gomigrate (migration_id) values (?)"
}

// MigrationLogDeleteSQL returns SQL for deleting a migration"
func (m MySQL) MigrationLogDeleteSQL() string {
	return "DELETE FROM gomigrate WHERE migration_id = ?"
}

// GetMigrationCommands return SQL commands
func (m MySQL) GetMigrationCommands(SQL string) []string {
	delimiter := ";"
	// we look at the first line of the migration for `delimiter foo`.
	// If found, we strip the line off, unquote the value, and use it as the delimiter
	if strings.HasPrefix(SQL, "delimiter ") {
		delimiterOffset := len("delimiter ")
		contentSplit := strings.SplitN(SQL[delimiterOffset:], "\n", 2)

		delimiter = strings.TrimSpace(contentSplit[0])

		if len(contentSplit) > 1 {
			SQL = contentSplit[1]
		} else {
			SQL = ""
		}

		delimiterUnquoted, err := strconv.Unquote(delimiter)
		if err == nil {
			delimiter = delimiterUnquoted
		}
	}

	return strings.Split(SQL, delimiter)
}

// Mariadb adapter
type Mariadb struct {
	MySQL
}

// SQLite3 adapter
type SQLite3 struct{}

// SelectMigrationTableSQL gets table names from db catalog
func (s SQLite3) SelectMigrationTableSQL() string {
	return "SELECT name FROM SQLite_master WHERE type = 'table' AND name = ?"
}

// CreateMigrationTableSQL gets SQL to create table for holding migrations
func (s SQLite3) CreateMigrationTableSQL() string {
	return `CREATE TABLE gomigrate (
  id INTEGER PRIMARY KEY,
  migration_id INTEGER NOT NULL UNIQUE
)`
}

// GetMigrationSQL gets migration SQL for given id
func (s SQLite3) GetMigrationSQL() string {
	return "SELECT migration_id FROM gomigrate WHERE migration_id = ?"
}

// MigrationLogInsertSQL gets insert SQL for migration
func (s SQLite3) MigrationLogInsertSQL() string {
	return "INSERT INTO gomigrate (migration_id) values (?)"
}

// MigrationLogDeleteSQL returns SQL for deleting a migration"
func (s SQLite3) MigrationLogDeleteSQL() string {
	return "DELETE FROM gomigrate WHERE migration_id = ?"
}

// GetMigrationCommands return SQL commands
func (s SQLite3) GetMigrationCommands(SQL string) []string {
	return []string{SQL}
}

// MsSQL adapter
type MsSQL struct{}

// SelectMigrationTableSQL gets table names from db catalog
func (m MsSQL) SelectMigrationTableSQL() string {
	return "SELECT table_name FROM information_schema.tables WHERE table_name = ?"
}

// CreateMigrationTableSQL gets SQL to create table for holding migrations
func (m MsSQL) CreateMigrationTableSQL() string {
	return `CREATE TABLE gomigrate (
                  id           INT          NOT NULL IDENTITY,
                  migration_id BIGINT       NOT NULL UNIQUE,
                  PRIMARY KEY (id)
                )`
}

// GetMigrationSQL gets migration SQL for given id
func (m MsSQL) GetMigrationSQL() string {
	return `SELECT migration_id FROM gomigrate WHERE migration_id = ?`
}

// MigrationLogInsertSQL gets insert SQL for migration
func (m MsSQL) MigrationLogInsertSQL() string {
	return "INSERT INTO gomigrate (migration_id) values (?)"
}

// MigrationLogDeleteSQL returns SQL for deleting a migration"
func (m MsSQL) MigrationLogDeleteSQL() string {
	return "DELETE FROM gomigrate WHERE migration_id = ?"
}

// GetMigrationCommands return SQL commands
func (m MsSQL) GetMigrationCommands(SQL string) []string {
	return []string{SQL}
}
