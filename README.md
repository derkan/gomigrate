# gomigrate

[![Build Status](https://travis-ci.org/derkan/gomigrate.svg?branch=master)](https://travis-ci.org/derkan/gomigrate)


A SQL database migration toolkit in Golang that supports migrations from multiple sources including in memory and files on disk.

## Supported databases

- PostgreSQL  
- CockroachDB
- MariaDB
- MySQL
- Sqlite3
- MSSQL

## Usage

First import the package:

```go
import "github.com/derkan/gomigrate"
```

Load Migrations from disk:
```go
m, err = gomigrate.MigrationsFromPath(path, logger)
if err != nil {
  // deal with error
}
```

Given a `database/sql` database connection to a PostgreSQL database, `db`,
and a directory to migration files, create a migrator:

```go
migrator, _ := gomigrate.NewMigratorWithMigrations(db, gomigrate.Postgres{}, m)
migrator.Logger = logger
```

You may also specify a specific logger to use at creation time supporting interface:
```go
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Fatalf(format string, v ...interface{})
}
```
;such as logrus:

```go
migrator, _ := gomigrate.NewMigratorWithLogger(db, gomigrate.Postgres{}, m, logrus.New())
```

To migrate the database, run:

```go
err := migrator.Migrate()
```

To rollback the last migration, run:

```go
err := migrator.Rollback()
```

## Migration files

Migration files need to follow a standard format and must be present
in the same directory. Given "up" and "down" steps for a migration,
create a file for each by following this template:

```
{{ id }}_{{ name }}_{{ "up" or "down" }}.sql
```

For a given migration, the `id` and `name` fields must be the same.
The id field is an integer that corresponds to the order in which
the migration should run relative to the other migrations.

`id` should not be `0` as that value is used for internal validations.

### Custom delimiter

By default SQL clauses are delimited with ";", you can set a new delimiter
by adding following as first line to migration sql file(for example set 
delimiter to `#`):
`delimiter #`

### Example

If I'm trying to add a "users" table to the database, I would create
the following two files:

#### 1_add_users_table_up.sql

```
CREATE TABLE users();
```

#### 1_add_users_table_down.sql
```
DROP TABLE users;
```

## Migrations from Memory
Migrations can also be embedded directly in your go code and passed into the Migrator.  This can be useful for testdata fixtures or using go-bindata to build fixture data into your go binary.

```go
	migrations := []*Migration{
		{
			ID:   100,
			Name: "FirstMigration",
			Up: `CREATE TABLE first_table (
				id INTEGER PRIMARY KEY
			)`,
			Down: `drop table "first_table"`,
		},
		{
			ID:   110,
			Name: "SecondMigration",
			Up: `CREATE TABLE second_table (
				id INTEGER PRIMARY KEY
			)`,
			Down: `drop table "second_table"`,
		},
	}
	migrator, _ := gomigrate.NewMigratorWithMigrations(db, gomigrate.Postgres{}, migrations)
	migrator.Migrate()
```

## Copyright

Copyright (c) 2014 David Huie. See LICENSE.txt for further details.
