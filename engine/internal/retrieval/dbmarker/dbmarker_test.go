package dbmarker

import (
	"os"
	"path"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestMarker_CreateAndGetConfig(t *testing.T) {
	t.Run("create config then read empty", func(t *testing.T) {
		m := NewMarker(t.TempDir())
		require.NoError(t, m.CreateConfig())

		cfg, err := m.GetConfig()
		require.NoError(t, err)
		assert.Empty(t, cfg.DataStateAt)
		assert.Empty(t, cfg.DataType)
	})

	t.Run("save config with data then read back", func(t *testing.T) {
		m := NewMarker(t.TempDir())
		require.NoError(t, m.CreateConfig())

		saved := &Config{DataStateAt: "2024-01-15 10:30:00", DataType: PhysicalDataType}
		require.NoError(t, m.SaveConfig(saved))

		cfg, err := m.GetConfig()
		require.NoError(t, err)
		assert.Equal(t, saved.DataStateAt, cfg.DataStateAt)
		assert.Equal(t, saved.DataType, cfg.DataType)
	})

	t.Run("overwrite existing config", func(t *testing.T) {
		m := NewMarker(t.TempDir())
		require.NoError(t, m.CreateConfig())

		require.NoError(t, m.SaveConfig(&Config{DataStateAt: "old", DataType: LogicalDataType}))
		require.NoError(t, m.SaveConfig(&Config{DataStateAt: "new", DataType: PhysicalDataType}))

		cfg, err := m.GetConfig()
		require.NoError(t, err)
		assert.Equal(t, "new", cfg.DataStateAt)
		assert.Equal(t, PhysicalDataType, cfg.DataType)
	})
}

func TestMarker_GetConfig_MissingFile(t *testing.T) {
	m := NewMarker(t.TempDir())

	_, err := m.GetConfig()
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestMarker_InitBranching(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewMarker(tmpDir)
	require.NoError(t, m.InitBranching())

	t.Run("branches directory exists", func(t *testing.T) {
		info, err := os.Stat(path.Join(tmpDir, configDir, refsDir, branchesDir))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("snapshots directory exists", func(t *testing.T) {
		info, err := os.Stat(path.Join(tmpDir, configDir, refsDir, snapshotsDir))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("HEAD file exists", func(t *testing.T) {
		info, err := os.Stat(path.Join(tmpDir, configDir, headFile))
		require.NoError(t, err)
		assert.False(t, info.IsDir())
	})
}

func TestMarker_InitMainBranch(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewMarker(tmpDir)
	require.NoError(t, m.InitBranching())

	snapshots := []SnapshotInfo{
		{ID: "snap1", Parent: "0000", CreatedAt: "2024-01-01 00:00:00", StateAt: "2024-01-01 00:00:00"},
		{ID: "snap2", Parent: "snap1", CreatedAt: "2024-01-02 00:00:00", StateAt: "2024-01-02 00:00:00"},
	}

	require.NoError(t, m.InitMainBranch(snapshots))

	t.Run("main branch directory exists", func(t *testing.T) {
		info, err := os.Stat(m.buildBranchName(mainBranch))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("HEAD points to last snapshot", func(t *testing.T) {
		data, err := os.ReadFile(path.Join(tmpDir, configDir, headFile))
		require.NoError(t, err)

		var h Head
		require.NoError(t, yaml.Unmarshal(data, &h))
		assert.Equal(t, buildSnapshotRef("snap2"), h.Ref)
	})

	t.Run("branch HEAD matches global HEAD", func(t *testing.T) {
		globalData, err := os.ReadFile(path.Join(tmpDir, configDir, headFile))
		require.NoError(t, err)

		branchData, err := os.ReadFile(m.buildBranchArtifactPath(mainBranch, headFile))
		require.NoError(t, err)

		assert.Equal(t, globalData, branchData)
	})

	t.Run("logs contain all snapshot entries", func(t *testing.T) {
		data, err := os.ReadFile(m.buildBranchArtifactPath(mainBranch, logsFile))
		require.NoError(t, err)
		content := string(data)

		assert.Contains(t, content, "0000 snap1 2024-01-01 00:00:00 2024-01-01 00:00:00")
		assert.Contains(t, content, "snap1 snap2 2024-01-02 00:00:00 2024-01-02 00:00:00")
	})

	t.Run("snapshot files are stored", func(t *testing.T) {
		for _, snap := range snapshots {
			data, err := os.ReadFile(m.buildSnapshotName(snap.ID))
			require.NoError(t, err)

			var info SnapshotInfo
			require.NoError(t, yaml.Unmarshal(data, &info))
			assert.Equal(t, snap.ID, info.ID)
			assert.Equal(t, snap.Parent, info.Parent)
		}
	})
}

func TestMarker_CreateBranch(t *testing.T) {
	m := setupMainBranch(t)

	require.NoError(t, m.CreateBranch("feature", mainBranch))

	t.Run("new branch directory exists", func(t *testing.T) {
		info, err := os.Stat(m.buildBranchName("feature"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("new branch HEAD matches base branch HEAD", func(t *testing.T) {
		baseData, err := os.ReadFile(m.buildBranchArtifactPath(mainBranch, headFile))
		require.NoError(t, err)

		newData, err := os.ReadFile(m.buildBranchArtifactPath("feature", headFile))
		require.NoError(t, err)

		assert.Equal(t, baseData, newData)
	})
}

func TestMarker_CreateBranch_MissingBase(t *testing.T) {
	m := setupMainBranch(t)

	err := m.CreateBranch("feature", "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read file")
}

func TestMarker_ListBranches(t *testing.T) {
	m := setupMainBranch(t)

	t.Run("single main branch", func(t *testing.T) {
		branches, err := m.ListBranches()
		require.NoError(t, err)
		assert.Equal(t, []string{"main"}, branches)
	})

	t.Run("multiple branches", func(t *testing.T) {
		require.NoError(t, m.CreateBranch("alpha", mainBranch))
		require.NoError(t, m.CreateBranch("beta", mainBranch))

		branches, err := m.ListBranches()
		require.NoError(t, err)
		sort.Strings(branches)
		assert.Equal(t, []string{"alpha", "beta", "main"}, branches)
	})
}

func TestMarker_ListBranches_MissingDir(t *testing.T) {
	m := NewMarker(t.TempDir())

	_, err := m.ListBranches()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read repository")
}

func TestMarker_GetSnapshotID(t *testing.T) {
	m := setupMainBranch(t)

	snapshotID, err := m.GetSnapshotID(mainBranch)
	require.NoError(t, err)
	assert.Equal(t, "snap2", snapshotID)
}

func TestMarker_GetSnapshotID_MissingBranch(t *testing.T) {
	m := setupMainBranch(t)

	_, err := m.GetSnapshotID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read file")
}

func TestMarker_SaveSnapshotRef(t *testing.T) {
	m := setupMainBranch(t)
	require.NoError(t, m.CreateBranch("feature", mainBranch))

	newSnap := SnapshotInfo{ID: "snap3", Parent: "snap2", CreatedAt: "2024-01-03 00:00:00", StateAt: "2024-01-03 00:00:00"}
	require.NoError(t, m.storeSnapshotInfo(newSnap))

	require.NoError(t, m.SaveSnapshotRef("feature", "snap3"))

	snapshotID, err := m.GetSnapshotID("feature")
	require.NoError(t, err)
	assert.Equal(t, "snap3", snapshotID)

	t.Run("main branch unchanged", func(t *testing.T) {
		mainID, err := m.GetSnapshotID(mainBranch)
		require.NoError(t, err)
		assert.Equal(t, "snap2", mainID)
	})
}

func TestMarker_SaveSnapshotRef_MissingBranch(t *testing.T) {
	m := setupMainBranch(t)

	err := m.SaveSnapshotRef("nonexistent", "snap3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read file")
}

func setupMainBranch(t *testing.T) *Marker {
	t.Helper()

	m := NewMarker(t.TempDir())
	require.NoError(t, m.InitBranching())

	snapshots := []SnapshotInfo{
		{ID: "snap1", Parent: "0000", CreatedAt: "2024-01-01 00:00:00", StateAt: "2024-01-01 00:00:00"},
		{ID: "snap2", Parent: "snap1", CreatedAt: "2024-01-02 00:00:00", StateAt: "2024-01-02 00:00:00"},
	}

	require.NoError(t, m.InitMainBranch(snapshots))

	return m
}
