// A simple database migrator for PostgreSQL.

package gomigrate

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"sort"
)

type migrationType string

const (
	migrationTableName = "gomigrate"
	upMigration        = migrationType("up")
	downMigration      = migrationType("down")
)

var (
	InvalidMigrationFile  = errors.New("Invalid migration file")
	InvalidMigrationPair  = errors.New("Invalid pair of migration files")
	InvalidMigrationType  = errors.New("Invalid migration type")
	ErrDuplicateMigration = errors.New("Duplicate migrations found")
)

// Migrator contains the information needed to migrate a database schema.
type Migrator struct {
	DB             *sql.DB
	MigrationsPath string
	dbAdapter      Migratable
	migrations     map[uint64]*Migration
	Logger         Logger
}

// Logger represents the standard logging interface allows different logging
// implementations to be used.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Fatalf(format string, v ...interface{})
}

// MigrationTableExists returns true if the migration table already exists.
func (m *Migrator) MigrationTableExists() (bool, error) {
	row := m.DB.QueryRow(m.dbAdapter.SelectMigrationTableSql(), migrationTableName)
	var tableName string
	err := row.Scan(&tableName)
	if err == sql.ErrNoRows {
		m.Logger.Print("Migrations table not found")
		return false, nil
	}
	if err != nil {
		m.Logger.Printf("Error checking for migration table: %v", err)
		return false, err
	}
	m.Logger.Print("Migrations table found")
	return true, nil
}

// CreateMigrationsTable creates the migrations table if it doesn't exist.
func (m *Migrator) CreateMigrationsTable() error {
	_, err := m.DB.Exec(m.dbAdapter.CreateMigrationTableSql())
	if err != nil {
		m.Logger.Fatalf("Error creating migrations table: %v", err)
	}

	m.Logger.Printf("Created migrations table: %s", migrationTableName)

	return nil
}

// NewMigratorWithMigrations returns a new Migrator setup with the given
// migrations.  It validates the migrations (i.e. no duplicates) but doesn't
// connect to the database.  All changes happen in the Migrate() function.
func NewMigratorWithMigrations(db *sql.DB, adapter Migratable, migrations []*Migration) (*Migrator, error) {
	migrator := &Migrator{
		DB:         db,
		dbAdapter:  adapter,
		migrations: make(map[uint64]*Migration),
		Logger:     log.New(os.Stderr, "[gomigrate] ", log.LstdFlags),
	}
	for _, m := range migrations {
		m.Status = Inactive
		if ok := m.Validate(); ok != nil {
			return nil, ok
		}
		if _, ok := migrator.migrations[m.ID]; ok {
			return nil, ErrDuplicateMigration
		}
		migrator.migrations[m.ID] = m
	}
	return migrator, nil
}

// NewMigrator is the previous api for gomigrate.  It loads migrations from
// disk and return a new migrator.
func NewMigrator(db *sql.DB, adapter Migratable, migrationsPath string) (*Migrator, error) {
	return NewMigratorWithLogger(db, adapter, migrationsPath, log.New(os.Stderr, "[gomigrate] ", log.LstdFlags))
}

// NewMigratorWithLogger is the previous api for gomigrate.  It loads migrations from
// disk and you can provide a Logger object.
func NewMigratorWithLogger(db *sql.DB, adapter Migratable, migrationsPath string, logger Logger) (*Migrator, error) {
	migrations, err := MigrationsFromPath(migrationsPath, logger)
	if err != nil {
		return nil, err
	}
	m, err := NewMigratorWithMigrations(db, adapter, migrations)
	if err != nil {
		return nil, err
	}
	m.Logger = logger

	return m, nil
}

// Migrate runs the given migrations against the database.
// It will also create the migration meta table if needed and will only run
// migrations that haven't already been run.
func (m *Migrator) Migrate() error {
	// Create the migrations table if it doesn't exist.
	tableExists, err := m.MigrationTableExists()
	if err != nil {
		return err
	}
	if !tableExists {
		if err := m.CreateMigrationsTable(); err != nil {
			return err
		}
	}
	if err := m.getMigrationStatuses(); err != nil {
		return err
	}
	for _, migration := range m.Migrations(Inactive) {
		if err := m.ApplyMigration(migration, upMigration); err != nil {
			return err
		}
	}

	return nil
}

// Queries the migration table to determine the status of each
// migration.
func (m *Migrator) getMigrationStatuses() error {
	for _, migration := range m.migrations {
		row := m.DB.QueryRow(m.dbAdapter.GetMigrationSql(), migration.ID)
		var mid uint64
		err := row.Scan(&mid)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			m.Logger.Printf(
				"Error getting migration status for %s: %v",
				migration.Name,
				err,
			)
			return err
		}
		migration.Status = Active
	}
	return nil
}

// Migrations returns a sorted list of migration ids for a given status. -1 returns
// all migrations.
func (m *Migrator) Migrations(status int) []*Migration {
	// Sort all migration ids.
	var ids []uint64
	for id := range m.migrations {
		ids = append(ids, id)
	}
	sort.Sort(uint64slice(ids))

	// Find ids for the given status.
	var migrations []*Migration
	for _, id := range ids {
		migration := m.migrations[id]
		if status == -1 || migration.Status == status {
			migrations = append(migrations, migration)
		}
	}

	return migrations
}

// ApplyMigration applies a single migration in the given direction.
func (m *Migrator) ApplyMigration(migration *Migration, mType migrationType) error {
	m.Logger.Printf("Applying migration: %s", migration.Name)
	var sql string
	if mType == upMigration && migration.Up != "" {
		sql = migration.Up
	} else if mType == downMigration && migration.Down != "" {
		sql = migration.Down
	} else {
		return InvalidMigrationType
	}
	transaction, err := m.DB.Begin()
	if err != nil {
		m.Logger.Printf("Error opening transaction: %v", err)
		return err
	}

	// Certain adapters can not handle multiple sql commands in one file so we need the adapter to split up the command
	commands := m.dbAdapter.GetMigrationCommands(string(sql))

	// Perform the migration.
	for _, cmd := range commands {
		result, err := transaction.Exec(cmd)
		if err != nil {
			m.Logger.Printf("Error executing migration: %v", err)
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				m.Logger.Printf("Error rolling back transaction: %v", rollbackErr)
				return rollbackErr
			}
			return err
		}
		if result != nil {
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				m.Logger.Printf("Error getting rows affected: %v", err)
				if rollbackErr := transaction.Rollback(); rollbackErr != nil {
					m.Logger.Printf("Error rolling back transaction: %v", rollbackErr)
					return rollbackErr
				}
				return err
			}
			m.Logger.Printf("Rows affected: %v", rowsAffected)
		}
	}

	// Log the event.
	if mType == upMigration {
		_, err = transaction.Exec(
			m.dbAdapter.MigrationLogInsertSql(),
			migration.ID,
		)
	} else {
		_, err = transaction.Exec(
			m.dbAdapter.MigrationLogDeleteSql(),
			migration.ID,
		)
	}
	if err != nil {
		m.Logger.Printf("Error logging migration: %v", err)
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			m.Logger.Printf("Error rolling back transaction: %v", rollbackErr)
			return rollbackErr
		}
		return err
	}

	// Commit and update the struct status.
	if err := transaction.Commit(); err != nil {
		m.Logger.Printf("Error commiting transaction: %v", err)
		return err
	}
	if mType == upMigration {
		migration.Status = Active
	} else {
		migration.Status = Inactive
	}

	return nil
}

// Rollback rolls back the last migration.
func (m *Migrator) Rollback() error {
	return m.RollbackN(1)
}

// RollbackN rolls back N migrations.
func (m *Migrator) RollbackN(n int) error {
	// checks the database for migration statuses
	if err := m.getMigrationStatuses(); err != nil {
		return err
	}

	migrations := m.Migrations(Active)
	if len(migrations) == 0 {
		return nil
	}

	lastMigration := len(migrations) - 1 - n

	for i := len(migrations) - 1; i != lastMigration; i-- {
		if err := m.ApplyMigration(migrations[i], downMigration); err != nil {
			return err
		}
	}

	return nil
}

// RollbackAll rolls back all migrations.
func (m *Migrator) RollbackAll() error {
	migrations := m.Migrations(Active)
	return m.RollbackN(len(migrations))
}
