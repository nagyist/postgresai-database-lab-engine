package pgconfig

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/engine/postgres/tools/defaults"
)

func TestReadConfig(t *testing.T) {
	controlData := `
restore_command = 'wal-g wal-fetch %f %p'
standby_mode = 'on'
recovery_target_timeline = latest
`
	expected := map[string]string{
		"restore_command":          "wal-g wal-fetch %f %p",
		"standby_mode":             "on",
		"recovery_target_timeline": "latest",
	}

	f, err := os.CreateTemp("", "readPGConfig*")
	require.Nil(t, err)
	defer func() { _ = os.Remove(f.Name()) }()

	err = os.WriteFile(f.Name(), []byte(controlData), 0644)
	require.Nil(t, err)

	fileConfig, err := readConfig(f.Name())
	require.Nil(t, err)

	assert.Equal(t, len(expected), len(fileConfig))
	assert.Equal(t, expected["restore_command"], fileConfig["restore_command"])
	assert.Equal(t, expected["standby_mode"], fileConfig["standby_mode"])
	assert.Equal(t, expected["recovery_target_timeline"], fileConfig["recovery_target_timeline"])
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name       string
		dataDir    string
		configName string
		expected   string
	}{
		{name: "standard config", dataDir: "/pgdata", configName: "postgresql.conf", expected: "/pgdata/postgresql.dblab.postgresql.conf"},
		{name: "sync config", dataDir: "/pgdata", configName: "sync.conf", expected: "/pgdata/postgresql.dblab.sync.conf"},
		{name: "nested data dir", dataDir: "/var/lib/postgresql/data", configName: "user_defined.conf", expected: "/var/lib/postgresql/data/postgresql.dblab.user_defined.conf"},
		{name: "empty data dir", dataDir: "", configName: "pg_control.conf", expected: "postgresql.dblab.pg_control.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetConfigPath(tt.dataDir, tt.configName))
		})
	}
}

func TestManager_recoveryFilename(t *testing.T) {
	tests := []struct {
		name      string
		pgVersion float64
		expected  string
	}{
		{name: "pg 11 uses plain recovery.conf", pgVersion: 11, expected: "recovery.conf"},
		{name: "pg 10 uses plain recovery.conf", pgVersion: 10, expected: "recovery.conf"},
		{name: "pg 12 uses prefixed recovery.conf", pgVersion: defaults.PGVersion12, expected: "postgresql.dblab.recovery.conf"},
		{name: "pg 13 uses prefixed recovery.conf", pgVersion: 13, expected: "postgresql.dblab.recovery.conf"},
		{name: "pg 16 uses prefixed recovery.conf", pgVersion: 16, expected: "postgresql.dblab.recovery.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{pgVersion: tt.pgVersion, dataDir: "/pgdata"}
			assert.Equal(t, tt.expected, m.recoveryFilename())
		})
	}
}

func TestManager_recoveryPath(t *testing.T) {
	tests := []struct {
		name      string
		pgVersion float64
		dataDir   string
		expected  string
	}{
		{name: "pg 11", pgVersion: 11, dataDir: "/pgdata", expected: "/pgdata/recovery.conf"},
		{name: "pg 14", pgVersion: 14, dataDir: "/pgdata", expected: "/pgdata/postgresql.dblab.recovery.conf"},
		{name: "pg 12 nested dir", pgVersion: 12, dataDir: "/var/lib/pg/data", expected: "/var/lib/pg/data/postgresql.dblab.recovery.conf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manager{pgVersion: tt.pgVersion, dataDir: tt.dataDir}
			assert.Equal(t, tt.expected, m.recoveryPath())
		})
	}
}

func TestManager_standbySignalPath(t *testing.T) {
	m := &Manager{dataDir: "/pgdata"}
	assert.Equal(t, "/pgdata/standby.signal", m.standbySignalPath())

	m2 := &Manager{dataDir: "/var/lib/postgresql/data"}
	assert.Equal(t, "/var/lib/postgresql/data/standby.signal", m2.standbySignalPath())
}

func TestManager_recoverySignalPath(t *testing.T) {
	m := &Manager{dataDir: "/pgdata"}
	assert.Equal(t, "/pgdata/recovery.signal", m.recoverySignalPath())

	m2 := &Manager{dataDir: "/var/lib/postgresql/data"}
	assert.Equal(t, "/var/lib/postgresql/data/recovery.signal", m2.recoverySignalPath())
}

func TestManager_isInitialized(t *testing.T) {
	t.Run("file does not exist returns false", func(t *testing.T) {
		m := &Manager{dataDir: t.TempDir()}
		result, err := m.isInitialized()
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("empty file returns false", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(path.Join(dir, PgConfName), []byte{}, 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		result, err := m.isInitialized()
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("file with initialization marker returns true", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(path.Join(dir, PgConfName), []byte(initializedLabel+"\nsome config"), 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		result, err := m.isInitialized()
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("file without marker returns false", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(path.Join(dir, PgConfName), []byte("shared_buffers = '256MB'\nlisten_addresses = '*'"), 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		result, err := m.isInitialized()
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("marker on second line returns false", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(path.Join(dir, PgConfName), []byte("some config\n"+initializedLabel), 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		result, err := m.isInitialized()
		require.NoError(t, err)
		assert.False(t, result)
	})
}

func TestManager_getConfigPath(t *testing.T) {
	m := &Manager{dataDir: "/pgdata"}
	assert.Equal(t, "/pgdata/postgresql.dblab.promotion.conf", m.getConfigPath(promotionConfigName))
	assert.Equal(t, "/pgdata/postgresql.dblab.snapshot.conf", m.getConfigPath(snapshotConfigName))
	assert.Equal(t, "/pgdata/postgresql.dblab.user_defined.conf", m.getConfigPath(userConfigName))
}

func TestReadConfig_nonexistentFile(t *testing.T) {
	cfg, err := readConfig("/nonexistent/path/to/config.conf")
	require.NoError(t, err)
	assert.Empty(t, cfg)
}

func TestReadConfig_linesWithoutEquals(t *testing.T) {
	dir := t.TempDir()
	cfgPath := path.Join(dir, "test.conf")
	content := "# this is a comment\nvalid_key = 'value'\nempty line\nanother_key = another_value"
	err := os.WriteFile(cfgPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := readConfig(cfgPath)
	require.NoError(t, err)
	assert.Len(t, cfg, 2)
	assert.Equal(t, "value", cfg["valid_key"])
	assert.Equal(t, "another_value", cfg["another_key"])
}

func TestReadConfig_valueWithEquals(t *testing.T) {
	dir := t.TempDir()
	cfgPath := path.Join(dir, "test.conf")
	content := "restore_command = 'pg_basebackup -D %p --source=host=primary'"
	err := os.WriteFile(cfgPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := readConfig(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "pg_basebackup -D %p --source=host=primary", cfg["restore_command"])
}

func TestManager_rewriteConfig(t *testing.T) {
	t.Run("writes key-value pairs to file", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := path.Join(dir, "test.conf")
		m := &Manager{dataDir: dir}

		err := m.rewriteConfig(cfgPath, map[string]string{"shared_buffers": "256MB", "work_mem": "64MB"})
		require.NoError(t, err)

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)

		text := string(content)
		assert.Contains(t, text, "shared_buffers = '256MB'")
		assert.Contains(t, text, "work_mem = '64MB'")
	})

	t.Run("overwrites existing content", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := path.Join(dir, "test.conf")
		err := os.WriteFile(cfgPath, []byte("old_param = 'old_value'"), 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		err = m.rewriteConfig(cfgPath, map[string]string{"new_param": "new_value"})
		require.NoError(t, err)

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)

		text := string(content)
		assert.NotContains(t, text, "old_param")
		assert.Contains(t, text, "new_param = 'new_value'")
	})

	t.Run("empty config writes empty file", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := path.Join(dir, "test.conf")
		m := &Manager{dataDir: dir}

		err := m.rewriteConfig(cfgPath, map[string]string{})
		require.NoError(t, err)

		content, err := os.ReadFile(cfgPath)
		require.NoError(t, err)
		assert.Empty(t, string(content))
	})
}

func TestManager_truncateConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := path.Join(dir, "test.conf")
	err := os.WriteFile(cfgPath, []byte("shared_buffers = '256MB'\nwork_mem = '64MB'"), 0644)
	require.NoError(t, err)

	m := &Manager{dataDir: dir}
	err = m.truncateConfig(cfgPath)
	require.NoError(t, err)

	content, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestManager_rewritePostgresConfig(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{dataDir: dir}

	err := m.rewritePostgresConfig()
	require.NoError(t, err)

	content, err := os.ReadFile(path.Join(dir, PgConfName))
	require.NoError(t, err)

	text := string(content)
	assert.True(t, len(text) > 0)
	assert.Contains(t, text, initializedLabel)
	assert.Contains(t, text, "include_if_exists postgresql.dblab.postgresql.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.sync.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.promotion.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.snapshot.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.user_defined.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.pg_control.conf")
	assert.Contains(t, text, "include_if_exists postgresql.dblab.recovery.conf")
}

func TestManager_removeOptionally(t *testing.T) {
	t.Run("removes existing file", func(t *testing.T) {
		dir := t.TempDir()
		filePath := path.Join(dir, "test.signal")
		err := os.WriteFile(filePath, []byte("content"), 0644)
		require.NoError(t, err)

		m := &Manager{dataDir: dir}
		err = m.removeOptionally(filePath)
		require.NoError(t, err)

		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("non-existent file returns no error", func(t *testing.T) {
		m := &Manager{dataDir: t.TempDir()}
		err := m.removeOptionally("/nonexistent/path/file.conf")
		assert.NoError(t, err)
	})
}

func TestReadUserConfig(t *testing.T) {
	t.Run("returns empty map when file does not exist", func(t *testing.T) {
		cfg, err := ReadUserConfig("/nonexistent/datadir")
		require.NoError(t, err)
		assert.Empty(t, cfg)
	})

	t.Run("reads user config from correct path", func(t *testing.T) {
		dir := t.TempDir()
		cfgPath := path.Join(dir, "postgresql.dblab.user_defined.conf")
		err := os.WriteFile(cfgPath, []byte("max_connections = '200'\nwork_mem = '128MB'"), 0644)
		require.NoError(t, err)

		cfg, err := ReadUserConfig(dir)
		require.NoError(t, err)
		assert.Equal(t, "200", cfg["max_connections"])
		assert.Equal(t, "128MB", cfg["work_mem"])
	})
}

func TestManager_ApplyRecovery(t *testing.T) {
	t.Run("empty config does nothing", func(t *testing.T) {
		m := &Manager{pgVersion: 14, dataDir: t.TempDir()}
		err := m.ApplyRecovery(map[string]string{})
		require.NoError(t, err)
	})

	t.Run("pg 14 creates standby signal and recovery config", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{pgVersion: 14, dataDir: dir}

		err := m.ApplyRecovery(map[string]string{"restore_command": "wal-g wal-fetch %f %p"})
		require.NoError(t, err)

		_, err = os.Stat(path.Join(dir, standbySignal))
		assert.NoError(t, err)

		content, err := os.ReadFile(m.recoveryPath())
		require.NoError(t, err)
		assert.Contains(t, string(content), "restore_command = 'wal-g wal-fetch %f %p'")
	})

	t.Run("pg 11 does not create standby signal", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{pgVersion: 11, dataDir: dir}

		err := m.ApplyRecovery(map[string]string{"standby_mode": "on"})
		require.NoError(t, err)

		_, err = os.Stat(path.Join(dir, standbySignal))
		assert.True(t, os.IsNotExist(err))

		content, err := os.ReadFile(m.recoveryPath())
		require.NoError(t, err)
		assert.Contains(t, string(content), "standby_mode = 'on'")
	})
}

func TestManager_ApplyPgControl(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	err := m.ApplyPgControl(map[string]string{"max_connections": "200", "wal_level": "replica"})
	require.NoError(t, err)

	content, err := os.ReadFile(m.getConfigPath(pgControlName))
	require.NoError(t, err)

	text := string(content)
	assert.Contains(t, text, "max_connections = '200'")
	assert.Contains(t, text, "wal_level = 'replica'")
}

func TestManager_ApplySync(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	err := m.ApplySync(map[string]string{"primary_conninfo": "host=primary port=5432"})
	require.NoError(t, err)

	content, err := os.ReadFile(m.getConfigPath(syncConfigName))
	require.NoError(t, err)
	assert.Contains(t, string(content), "primary_conninfo = 'host=primary port=5432'")
}

func TestManager_TruncateRecoveryConfig(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	err := os.WriteFile(m.recoveryPath(), []byte("restore_command = 'some cmd'"), 0644)
	require.NoError(t, err)

	err = m.TruncateRecoveryConfig()
	require.NoError(t, err)

	content, err := os.ReadFile(m.recoveryPath())
	require.NoError(t, err)
	assert.Empty(t, content)
}

func TestManager_RemoveRecoveryConfig(t *testing.T) {
	t.Run("pg 14 removes recovery config and signal files", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{pgVersion: 14, dataDir: dir}

		require.NoError(t, os.WriteFile(m.recoveryPath(), []byte("data"), 0644))
		require.NoError(t, os.WriteFile(m.standbySignalPath(), []byte(""), 0644))
		require.NoError(t, os.WriteFile(m.recoverySignalPath(), []byte(""), 0644))

		err := m.RemoveRecoveryConfig()
		require.NoError(t, err)

		_, err = os.Stat(m.recoveryPath())
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(m.standbySignalPath())
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(m.recoverySignalPath())
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("pg 11 only removes recovery config", func(t *testing.T) {
		dir := t.TempDir()
		m := &Manager{pgVersion: 11, dataDir: dir}

		require.NoError(t, os.WriteFile(m.recoveryPath(), []byte("data"), 0644))

		err := m.RemoveRecoveryConfig()
		require.NoError(t, err)

		_, err = os.Stat(m.recoveryPath())
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("no error when files do not exist", func(t *testing.T) {
		m := &Manager{pgVersion: 14, dataDir: t.TempDir()}
		err := m.RemoveRecoveryConfig()
		assert.NoError(t, err)
	})
}

func TestManager_ApplyPromotion(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	err := m.ApplyPromotion(map[string]string{"hot_standby": "off"})
	require.NoError(t, err)

	content, err := os.ReadFile(m.getConfigPath(promotionConfigName))
	require.NoError(t, err)
	assert.Contains(t, string(content), "hot_standby = 'off'")
}

func TestManager_ApplySnapshot(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	err := m.ApplySnapshot(map[string]string{"shared_buffers": "512MB"})
	require.NoError(t, err)

	content, err := os.ReadFile(m.getConfigPath(snapshotConfigName))
	require.NoError(t, err)
	assert.Contains(t, string(content), "shared_buffers = '512MB'")
}

func TestManager_AppendGeneralConfig(t *testing.T) {
	dir := t.TempDir()
	m := &Manager{pgVersion: 14, dataDir: dir}

	cfgPath := m.getConfigPath(PgConfName)
	err := os.WriteFile(cfgPath, []byte("existing_param = 'value'"), 0644)
	require.NoError(t, err)

	err = m.AppendGeneralConfig(map[string]string{"new_param": "new_value"})
	require.NoError(t, err)

	content, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	text := string(content)
	assert.Contains(t, text, "existing_param = 'value'")
	assert.Contains(t, text, "new_param = 'new_value'")
}
