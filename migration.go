// Holds metadata about a migration.

package gomigrate

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Migration statuses.
const (
	Inactive = iota
	Active
)

// Migration holds configuration information for a given migration.
type Migration struct {
	ID     uint64
	Name   string
	Status int
	Up     string
	Down   string
	Source string
}

// Validate checks that a migration is properly formed and named.
func (m *Migration) Validate() error {
	if m.ID == 0 {
		return &ErrInvalidMigration{
			ID:   m.ID,
			Name: m.Name,
			Err:  "Id can't be zero",
		}
	}
	if m.Name == "" {
		return &ErrInvalidMigration{
			ID:   m.ID,
			Name: m.Name,
			Err:  "Name can't be empty",
		}
	}
	return nil
}

// ErrInvalidMigration encapsulates the reasons why a migration is invalid.
type ErrInvalidMigration struct {
	ID   uint64
	Name string
	Err  string
}

func (e *ErrInvalidMigration) Error() string {
	if e == nil {
		return "nil"
	}

	return fmt.Sprintf("Invalid Migration ID:%d, Name:'%s': %s", e.ID, e.Name, e.Err)
}

// MigrationsFromPath loads migrations from the given path.  Migration file
// naming and format requires two files per migration of the form:
// NUMBER_NAME_[UP|DOWN].sql
//
// Example:
//  1_add_users_table_up.sql
//  1_add_users_table_down.sql
//
// The name must match for each numbered pair.
func MigrationsFromPath(migrationsPath string, logger Logger) ([]*Migration, error) {
	// Normalize the migrations path.
	path := []byte(migrationsPath)
	pathLength := len(path)
	if path[pathLength-1] != '/' {
		path = append(path, '/')
	}

	logger.Printf("Migrations path: %s", path)
	migrations := map[uint64]*Migration{}

	pathGlob := append([]byte(path), []byte("*")...)
	matches, err := filepath.Glob(string(pathGlob))
	if err != nil {
		return nil, fmt.Errorf("Error while globbing migrations: %v", err)
	}

	for _, match := range matches {
		num, migrationType, name, err := parseMigrationPath(match)
		if err != nil {
			logger.Printf("Invalid migration file found: %s\n", match)
			continue
		}

		logger.Printf("Migration file found: %s\n", match)
		fileSQL, err := ioutil.ReadFile(match)
		if err != nil {
			logger.Printf("Error reading migration: %s", match)
			return nil, err
		}
		sql := string(fileSQL)

		if m, ok := migrations[num]; ok {
			m.Source = m.Source + " " + match
			if migrationType == upMigration {
				m.Up = sql
			} else {
				m.Down = sql
			}
		} else {
			migration := &Migration{
				ID:     num,
				Name:   name,
				Source: match,
				Status: Inactive,
			}
			if migrationType == upMigration {
				migration.Up = sql
			} else {
				migration.Down = sql
			}
			migrations[num] = migration
		}
	}

	// Validate each migration.
	for _, migration := range migrations {
		err = migration.Validate()
		if err != nil {
			logger.Printf("Invalid migration from files: %s\n", migration.Source)
			return nil, ErrInvalidMigrationPair
		}
	}

	logger.Printf("Migrations file pairs found: %v\n", len(migrations))

	v := make([]*Migration, 0, len(migrations))
	for _, value := range migrations {
		v = append(v, value)
	}

	return v, nil
}
