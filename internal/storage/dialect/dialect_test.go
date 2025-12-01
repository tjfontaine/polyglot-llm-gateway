package dialect

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		dialectType DialectType
		wantName    string
		wantErr     bool
	}{
		{"sqlite", SQLite, "sqlite", false},
		{"postgres", Postgres, "postgres", false},
		{"mysql", MySQL, "mysql", false},
		{"unknown", DialectType("unknown"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(tt.dialectType)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && d.Name() != tt.wantName {
				t.Errorf("Name() = %v, want %v", d.Name(), tt.wantName)
			}
		})
	}
}

func TestFromDriverName(t *testing.T) {
	tests := []struct {
		driverName string
		wantName   string
		wantErr    bool
	}{
		{"sqlite", "sqlite", false},
		{"sqlite3", "sqlite", false},
		{"postgres", "postgres", false},
		{"pgx", "postgres", false},
		{"mysql", "mysql", false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.driverName, func(t *testing.T) {
			d, err := FromDriverName(tt.driverName)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromDriverName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && d.Name() != tt.wantName {
				t.Errorf("Name() = %v, want %v", d.Name(), tt.wantName)
			}
		})
	}
}

func TestSQLiteDialect_Rebind(t *testing.T) {
	d := &sqliteDialect{}
	query := "SELECT * FROM users WHERE id = ? AND name = ?"
	got := d.Rebind(query)
	if got != query {
		t.Errorf("Rebind() = %v, want %v", got, query)
	}
}

func TestPostgresDialect_Rebind(t *testing.T) {
	d := &postgresDialect{}
	tests := []struct {
		query string
		want  string
	}{
		{"SELECT * FROM users WHERE id = ?", "SELECT * FROM users WHERE id = $1"},
		{"SELECT * FROM users WHERE id = ? AND name = ?", "SELECT * FROM users WHERE id = $1 AND name = $2"},
		{"INSERT INTO users VALUES (?, ?, ?)", "INSERT INTO users VALUES ($1, $2, $3)"},
		{"SELECT * FROM users", "SELECT * FROM users"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := d.Rebind(tt.query)
			if got != tt.want {
				t.Errorf("Rebind() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMySQLDialect_Rebind(t *testing.T) {
	d := &mysqlDialect{}
	query := "SELECT * FROM users WHERE id = ? AND name = ?"
	got := d.Rebind(query)
	if got != query {
		t.Errorf("Rebind() = %v, want %v", got, query)
	}
}

func TestSQLiteDialect_UpsertClause(t *testing.T) {
	d := &sqliteDialect{}

	got := d.UpsertClause("id", nil)
	want := "ON CONFLICT(id) DO NOTHING"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}

	got = d.UpsertClause("id", []string{"name", "updated_at"})
	want = "ON CONFLICT(id) DO UPDATE SET name=excluded.name, updated_at=excluded.updated_at"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}
}

func TestPostgresDialect_UpsertClause(t *testing.T) {
	d := &postgresDialect{}

	got := d.UpsertClause("id", nil)
	want := "ON CONFLICT (id) DO NOTHING"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}

	got = d.UpsertClause("id", []string{"name", "updated_at"})
	want = "ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}
}

func TestMySQLDialect_UpsertClause(t *testing.T) {
	d := &mysqlDialect{}

	got := d.UpsertClause("id", nil)
	want := "ON DUPLICATE KEY UPDATE id = id"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}

	got = d.UpsertClause("id", []string{"name", "updated_at"})
	want = "ON DUPLICATE KEY UPDATE name = VALUES(name), updated_at = VALUES(updated_at)"
	if got != want {
		t.Errorf("UpsertClause() = %v, want %v", got, want)
	}
}

func TestDialect_Types(t *testing.T) {
	tests := []struct {
		name          string
		dialect       Dialect
		boolType      string
		timestampType string
		textType      string
	}{
		{"sqlite", &sqliteDialect{}, "INTEGER", "TIMESTAMP", "TEXT"},
		{"postgres", &postgresDialect{}, "BOOLEAN", "TIMESTAMP WITH TIME ZONE", "TEXT"},
		{"mysql", &mysqlDialect{}, "TINYINT(1)", "DATETIME(6)", "LONGTEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dialect.BooleanType(); got != tt.boolType {
				t.Errorf("BooleanType() = %v, want %v", got, tt.boolType)
			}
			if got := tt.dialect.TimestampType(); got != tt.timestampType {
				t.Errorf("TimestampType() = %v, want %v", got, tt.timestampType)
			}
			if got := tt.dialect.TextType(); got != tt.textType {
				t.Errorf("TextType() = %v, want %v", got, tt.textType)
			}
		})
	}
}

func TestDialect_SupportsReturning(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		want    bool
	}{
		{"sqlite", &sqliteDialect{}, true},
		{"postgres", &postgresDialect{}, true},
		{"mysql", &mysqlDialect{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.dialect.SupportsReturning(); got != tt.want {
				t.Errorf("SupportsReturning() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDialect_PragmaStatements(t *testing.T) {
	sqliteD := &sqliteDialect{}
	pragmas := sqliteD.PragmaStatements()
	if len(pragmas) == 0 {
		t.Error("SQLite should have pragma statements")
	}

	pgD := &postgresDialect{}
	if pgD.PragmaStatements() != nil {
		t.Error("PostgreSQL should not have pragma statements")
	}

	mysqlD := &mysqlDialect{}
	if mysqlD.PragmaStatements() != nil {
		t.Error("MySQL should not have pragma statements")
	}
}
