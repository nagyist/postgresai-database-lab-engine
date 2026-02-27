package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDatabaseRenames(t *testing.T) {
	tests := []struct {
		name    string
		renames map[string]string
		connDB  string
	}{
		{name: "empty map", renames: map[string]string{}, connDB: "postgres"},
		{name: "single valid rename", renames: map[string]string{"prod_db": "dblab_db"}, connDB: "postgres"},
		{name: "multiple valid renames", renames: map[string]string{"db1": "db1_new", "db2": "db2_new"}, connDB: "postgres"},
		{name: "hyphenated names", renames: map[string]string{"my-db": "my-new-db"}, connDB: "postgres"},
		{name: "underscore prefix", renames: map[string]string{"_private": "_renamed"}, connDB: "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, validateDatabaseRenames(tt.renames, tt.connDB))
		})
	}
}

func TestValidateDatabaseRenames_errors(t *testing.T) {
	tests := []struct {
		name    string
		renames map[string]string
		connDB  string
		errMsg  string
	}{
		{name: "empty old name", renames: map[string]string{"": "new_db"}, connDB: "postgres", errMsg: "must not be empty"},
		{name: "empty new name", renames: map[string]string{"old_db": ""}, connDB: "postgres", errMsg: "must not be empty"},
		{name: "rename source is connDB", renames: map[string]string{"postgres": "pg_renamed"}, connDB: "postgres", errMsg: "connection database"},
		{name: "target is connDB", renames: map[string]string{"mydb": "postgres"}, connDB: "postgres", errMsg: "target name is the connection database"},
		{name: "self rename", renames: map[string]string{"mydb": "mydb"}, connDB: "postgres", errMsg: "to itself"},
		{name: "duplicate targets", renames: map[string]string{"db1": "same_name", "db2": "same_name"}, connDB: "postgres", errMsg: "duplicate rename target"},
		{name: "chained rename", renames: map[string]string{"a": "b", "b": "c"}, connDB: "postgres", errMsg: "chained rename conflict"},
		{name: "name with double quote", renames: map[string]string{`db"inject`: "target"}, connDB: "postgres", errMsg: "must start with a letter"},
		{name: "name starting with digit", renames: map[string]string{"1db": "target"}, connDB: "postgres", errMsg: "must start with a letter"},
		{name: "name with spaces", renames: map[string]string{"my db": "target"}, connDB: "postgres", errMsg: "must start with a letter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDatabaseRenames(tt.renames, tt.connDB)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestBuildRenameCommand(t *testing.T) {
	tests := []struct {
		name     string
		username string
		connDB   string
		oldName  string
		newName  string
		expected []string
	}{
		{
			name: "simple rename", username: "postgres", connDB: "postgres", oldName: "prod_db", newName: "dblab_db",
			expected: []string{"psql", "-U", "postgres", "-d", "postgres", "-XAtc", `ALTER DATABASE "prod_db" RENAME TO "dblab_db"`},
		},
		{
			name: "hyphenated name", username: "admin", connDB: "management", oldName: "my-db", newName: "my_db",
			expected: []string{"psql", "-U", "admin", "-d", "management", "-XAtc", `ALTER DATABASE "my-db" RENAME TO "my_db"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRenameCommand(tt.username, tt.connDB, tt.oldName, tt.newName)
			assert.Equal(t, tt.expected, result)
		})
	}
}
