package physical

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCustomRecoveryConfig(t *testing.T) {
	customTool := newCustomTool(customOptions{
		RestoreCommand: "pg_basebackup -X stream -D dataDirectory",
	})

	recoveryConfig := customTool.GetRecoveryConfig(11.7)
	expectedResponse11 := map[string]string{
		"restore_command":          "pg_basebackup -X stream -D dataDirectory",
		"recovery_target_timeline": "latest",
	}
	assert.Equal(t, expectedResponse11, recoveryConfig)

	recoveryConfig = customTool.GetRecoveryConfig(12.3)
	expectedResponse12 := map[string]string{
		"restore_command": "pg_basebackup -X stream -D dataDirectory",
	}
	assert.Equal(t, expectedResponse12, recoveryConfig)
}

func TestCustomGetRestoreCommand(t *testing.T) {
	testCases := []struct {
		name     string
		command  string
		expected string
	}{
		{name: "simple command", command: "rsync -av /src /dst", expected: "rsync -av /src /dst"},
		{name: "empty command", command: "", expected: ""},
		{name: "complex command", command: "pg_basebackup -h host -U user -D /data -X stream", expected: "pg_basebackup -h host -U user -D /data -X stream"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := newCustomTool(customOptions{Command: tc.command})
			assert.Equal(t, tc.expected, c.GetRestoreCommand())
		})
	}
}

func TestCustomRecoveryConfig_EmptyRestoreCommand(t *testing.T) {
	c := newCustomTool(customOptions{RestoreCommand: ""})

	t.Run("pg11 empty restore command", func(t *testing.T) {
		cfg := c.GetRecoveryConfig(11.0)
		assert.Empty(t, cfg)
	})

	t.Run("pg12 empty restore command", func(t *testing.T) {
		cfg := c.GetRecoveryConfig(12.0)
		assert.Empty(t, cfg)
	})
}

func TestCustomRecoveryConfig_VersionBoundary(t *testing.T) {
	c := newCustomTool(customOptions{RestoreCommand: "some_command"})

	testCases := []struct {
		name           string
		version        float64
		expectTimeline bool
	}{
		{name: "pg 9.6", version: 9.6, expectTimeline: true},
		{name: "pg 11.99", version: 11.99, expectTimeline: true},
		{name: "pg 12.0 exact", version: 12.0, expectTimeline: false},
		{name: "pg 13.0", version: 13.0, expectTimeline: false},
		{name: "pg 15.0", version: 15.0, expectTimeline: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := c.GetRecoveryConfig(tc.version)
			if tc.expectTimeline {
				assert.Equal(t, "latest", cfg["recovery_target_timeline"])
			} else {
				_, exists := cfg["recovery_target_timeline"]
				assert.False(t, exists)
			}
			assert.Equal(t, "some_command", cfg["restore_command"])
		})
	}
}
