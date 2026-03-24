/*
2021 © Postgres.ai
*/

package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConnStrWithSpecialChars(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		username string
		password string
		expected string
	}{
		{name: "password with equals sign", host: "localhost", username: "postgres", password: "pass=word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass=word'"},
		{name: "password with spaces", host: "localhost", username: "postgres", password: "pass word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass word'"},
		{name: "password with single quote", host: "localhost", username: "postgres", password: "pass'word", expected: "host='localhost' port=5432 user='postgres' database='testdb' password='pass''word'"},
		{name: "password with backslash", host: "localhost", username: "postgres", password: `pass\word`, expected: `host='localhost' port=5432 user='postgres' database='testdb' password='pass\\word'`},
		{name: "combined quote and backslash in password", host: "localhost", username: "postgres", password: `test\'name`, expected: `host='localhost' port=5432 user='postgres' database='testdb' password='test\\''name'`},
		{name: "host with single quote", host: "host'name", username: "postgres", password: "pass", expected: "host='host''name' port=5432 user='postgres' database='testdb' password='pass'"},
		{name: "host with backslash", host: `host\name`, username: "postgres", password: "pass", expected: `host='host\\name' port=5432 user='postgres' database='testdb' password='pass'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConnectionString(tt.host, "5432", tt.username, "testdb", tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEscapeLibpqValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no special chars", input: "simple", expected: "simple"},
		{name: "single quote", input: "it's", expected: "it''s"},
		{name: "backslash", input: `path\to`, expected: `path\\to`},
		{name: "combined quote and backslash", input: `test\'name`, expected: `test\\''name`},
		{name: "empty string", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeLibpqValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
