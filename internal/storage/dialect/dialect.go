// Package dialect provides database dialect abstractions for multi-database support.
package dialect

import (
	"fmt"
	"strings"
)

// Dialect represents a SQL database dialect.
type Dialect interface {
	// Name returns the dialect name (e.g., "sqlite", "postgres", "mysql")
	Name() string

	// DriverName returns the database/sql driver name to use
	DriverName() string

	// Rebind converts ? placeholders to the dialect's format.
	// For example, PostgreSQL uses $1, $2, etc.
	Rebind(query string) string

	// AutoIncrementClause returns the clause for auto-increment primary keys
	AutoIncrementClause() string

	// BooleanType returns the SQL type for boolean values
	BooleanType() string

	// TimestampType returns the SQL type for timestamps
	TimestampType() string

	// TextType returns the SQL type for large text fields
	TextType() string

	// UpsertClause returns the ON CONFLICT/ON DUPLICATE KEY clause for upserts
	UpsertClause(conflictColumn string, updateColumns []string) string

	// SupportsReturning returns true if the dialect supports RETURNING clause
	SupportsReturning() bool

	// PragmaStatements returns dialect-specific initialization statements (e.g., PRAGMA for SQLite)
	PragmaStatements() []string

	// ColumnExistsQuery returns a query to check if a column exists in a table
	ColumnExistsQuery() string

	// CurrentTimestamp returns the SQL expression for current timestamp
	CurrentTimestamp() string
}

// DialectType represents supported database types
type DialectType string

const (
	SQLite   DialectType = "sqlite"
	Postgres DialectType = "postgres"
	MySQL    DialectType = "mysql"
)

// New creates a new Dialect based on the dialect type
func New(dialectType DialectType) (Dialect, error) {
	switch dialectType {
	case SQLite:
		return &sqliteDialect{}, nil
	case Postgres:
		return &postgresDialect{}, nil
	case MySQL:
		return &mysqlDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", dialectType)
	}
}

// FromDriverName returns the dialect for a given driver name
func FromDriverName(driverName string) (Dialect, error) {
	switch strings.ToLower(driverName) {
	case "sqlite", "sqlite3":
		return &sqliteDialect{}, nil
	case "postgres", "pgx":
		return &postgresDialect{}, nil
	case "mysql":
		return &mysqlDialect{}, nil
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driverName)
	}
}

// sqliteDialect implements Dialect for SQLite
type sqliteDialect struct{}

func (d *sqliteDialect) Name() string {
	return "sqlite"
}

func (d *sqliteDialect) DriverName() string {
	return "sqlite"
}

func (d *sqliteDialect) Rebind(query string) string {
	return query // SQLite uses ?
}

func (d *sqliteDialect) AutoIncrementClause() string {
	return "INTEGER PRIMARY KEY AUTOINCREMENT"
}

func (d *sqliteDialect) BooleanType() string {
	return "INTEGER"
}

func (d *sqliteDialect) TimestampType() string {
	return "TIMESTAMP"
}

func (d *sqliteDialect) TextType() string {
	return "TEXT"
}

func (d *sqliteDialect) SupportsReturning() bool {
	return true // SQLite 3.35+ supports RETURNING
}

func (d *sqliteDialect) CurrentTimestamp() string {
	return "CURRENT_TIMESTAMP"
}

func (d *sqliteDialect) UpsertClause(conflictColumn string, updateColumns []string) string {
	if len(updateColumns) == 0 {
		return fmt.Sprintf("ON CONFLICT(%s) DO NOTHING", conflictColumn)
	}
	updates := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		updates[i] = fmt.Sprintf("%s=excluded.%s", col, col)
	}
	return fmt.Sprintf("ON CONFLICT(%s) DO UPDATE SET %s", conflictColumn, strings.Join(updates, ", "))
}

func (d *sqliteDialect) PragmaStatements() []string {
	return []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	}
}

func (d *sqliteDialect) ColumnExistsQuery() string {
	return `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`
}

// postgresDialect implements Dialect for PostgreSQL
type postgresDialect struct{}

func (d *postgresDialect) Name() string {
	return "postgres"
}

func (d *postgresDialect) DriverName() string {
	return "pgx"
}

func (d *postgresDialect) Rebind(query string) string {
	// Convert ? placeholders to $1, $2, etc.
	var result strings.Builder
	idx := 1
	for _, ch := range query {
		if ch == '?' {
			result.WriteString(fmt.Sprintf("$%d", idx))
			idx++
		} else {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func (d *postgresDialect) AutoIncrementClause() string {
	return "SERIAL PRIMARY KEY"
}

func (d *postgresDialect) BooleanType() string {
	return "BOOLEAN"
}

func (d *postgresDialect) TimestampType() string {
	return "TIMESTAMP WITH TIME ZONE"
}

func (d *postgresDialect) TextType() string {
	return "TEXT"
}

func (d *postgresDialect) SupportsReturning() bool {
	return true
}

func (d *postgresDialect) CurrentTimestamp() string {
	return "NOW()"
}

func (d *postgresDialect) UpsertClause(conflictColumn string, updateColumns []string) string {
	if len(updateColumns) == 0 {
		return fmt.Sprintf("ON CONFLICT (%s) DO NOTHING", conflictColumn)
	}
	updates := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		updates[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
	}
	return fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s", conflictColumn, strings.Join(updates, ", "))
}

func (d *postgresDialect) PragmaStatements() []string {
	return nil // PostgreSQL doesn't use pragmas
}

func (d *postgresDialect) ColumnExistsQuery() string {
	return `SELECT COUNT(*) FROM information_schema.columns WHERE table_name = $1 AND column_name = $2`
}

// mysqlDialect implements Dialect for MySQL
type mysqlDialect struct{}

func (d *mysqlDialect) Name() string {
	return "mysql"
}

func (d *mysqlDialect) DriverName() string {
	return "mysql"
}

func (d *mysqlDialect) Rebind(query string) string {
	return query // MySQL uses ?
}

func (d *mysqlDialect) AutoIncrementClause() string {
	return "INT AUTO_INCREMENT PRIMARY KEY"
}

func (d *mysqlDialect) BooleanType() string {
	return "TINYINT(1)"
}

func (d *mysqlDialect) TimestampType() string {
	return "DATETIME(6)"
}

func (d *mysqlDialect) TextType() string {
	return "LONGTEXT"
}

func (d *mysqlDialect) SupportsReturning() bool {
	return false // MySQL doesn't support RETURNING
}

func (d *mysqlDialect) CurrentTimestamp() string {
	return "CURRENT_TIMESTAMP(6)"
}

func (d *mysqlDialect) UpsertClause(conflictColumn string, updateColumns []string) string {
	if len(updateColumns) == 0 {
		return "ON DUPLICATE KEY UPDATE id = id" // No-op update
	}
	updates := make([]string, len(updateColumns))
	for i, col := range updateColumns {
		updates[i] = fmt.Sprintf("%s = VALUES(%s)", col, col)
	}
	return "ON DUPLICATE KEY UPDATE " + strings.Join(updates, ", ")
}

func (d *mysqlDialect) PragmaStatements() []string {
	return nil // MySQL doesn't use pragmas
}

func (d *mysqlDialect) ColumnExistsQuery() string {
	return `SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ? AND table_schema = DATABASE()`
}
