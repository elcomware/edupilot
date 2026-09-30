package database_test

import (
	"io/fs"
	"strings"
	"testing"

	postgresmigrations "github.com/elcomware/edupilot/database/migrations/postgres"
	sqlitemigrations "github.com/elcomware/edupilot/database/migrations/sqlite"
)

// TestSchemaParity keeps the standalone (SQLite) and cloud (PostgreSQL)
// schemas in step. Divergence here is the classic source of "works on the site
// server, fails in the cloud" defects, so it is a test, not a review note.
func TestSchemaParity(t *testing.T) {
	sqliteSchema := parseSchema(t, sqlitemigrations.FS)
	postgresSchema := parseSchema(t, postgresmigrations.FS)

	// Guard against a silently empty parse, which would make this test lie.
	if len(sqliteSchema) < 15 || len(postgresSchema) < 15 {
		t.Fatalf("parsed %d SQLite and %d PostgreSQL tables; the parser is not reading the migrations",
			len(sqliteSchema), len(postgresSchema))
	}

	for table, columns := range sqliteSchema {
		other, ok := postgresSchema[table]
		if !ok {
			t.Errorf("table %q exists in SQLite but not in PostgreSQL", table)
			continue
		}
		for column := range columns {
			if !other[column] {
				t.Errorf("table %q: column %q exists in SQLite but not in PostgreSQL", table, column)
			}
		}
	}

	for table := range postgresSchema {
		if _, ok := sqliteSchema[table]; !ok {
			t.Errorf("table %q exists in PostgreSQL but not in SQLite", table)
		}
	}
}

func TestPostgresMigrationHasNoInvalidTypedDefaults(t *testing.T) {
	body, err := postgresmigrations.FS.ReadFile("0001_core_init.sql")
	if err != nil {
		t.Fatal(err)
	}

	for number, line := range strings.Split(string(body), "\n") {
		upper := strings.ToUpper(line)
		if !strings.Contains(upper, "DEFAULT ''") {
			continue
		}
		// An empty string is a valid literal for text-like columns only. A date
		// or timestamp default of '' is rejected by PostgreSQL.
		if strings.Contains(upper, "TEXT") {
			continue
		}
		for _, typed := range []string{"DATE", "TIMESTAMPTZ", "TIMESTAMP", "NUMERIC", "BIGINT", "INTEGER", "BOOLEAN"} {
			if strings.Contains(upper, typed) {
				t.Errorf("line %d: %s", number+1, strings.TrimSpace(line))
			}
		}
	}
}

// parseSchema extracts table and column names from CREATE TABLE statements.
// It reads the released migrations, not a live database, so it runs anywhere.
func parseSchema(t *testing.T, files fs.FS) map[string]map[string]bool {
	t.Helper()

	schema := map[string]map[string]bool{}

	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		body, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			t.Fatal(err)
		}

		for table, block := range createTableBlocks(string(body)) {
			columns := schema[table]
			if columns == nil {
				columns = map[string]bool{}
				schema[table] = columns
			}
			for _, line := range strings.Split(block, "\n") {
				columns[columnName(line)] = true
			}
		}
	}

	for table, columns := range schema {
		delete(columns, "")
		if len(columns) == 0 {
			t.Errorf("table %q has no columns", table)
		}
	}

	return schema
}

func createTableBlocks(sql string) map[string]string {
	blocks := map[string]string{}
	lines := strings.Split(sql, "\n")

	for index := 0; index < len(lines); index++ {
		fields := strings.Fields(lines[index])
		if len(fields) < 4 || !strings.EqualFold(fields[0], "CREATE") || !strings.EqualFold(fields[1], "TABLE") {
			continue
		}

		// The table name is the token before the opening parenthesis, which
		// skips over an optional IF NOT EXISTS.
		open := len(fields)
		for position, field := range fields {
			if strings.HasPrefix(field, "(") {
				open = position
				break
			}
		}
		if open == 0 || open > len(fields) {
			continue
		}

		table := strings.Trim(fields[open-1], `"`)
		switch strings.ToUpper(table) {
		case "EXISTS", "IF", "NOT":
			continue
		}

		var body strings.Builder
		for index++; index < len(lines); index++ {
			trimmed := strings.TrimSpace(lines[index])
			if trimmed == ");" || strings.HasPrefix(trimmed, ");") {
				break
			}
			body.WriteString(lines[index])
			body.WriteString("\n")
		}
		blocks[table] = body.String()
	}

	return blocks
}

func columnName(line string) string {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return ""
	}

	switch strings.ToUpper(fields[0]) {
	case "CONSTRAINT", "PRIMARY", "UNIQUE", "FOREIGN", "CHECK", ")", "--":
		return ""
	}
	return strings.Trim(fields[0], `"`)
}
