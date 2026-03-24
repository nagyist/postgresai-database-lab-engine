package diagnostic

import (
	"archive/tar"
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsCleanup(t *testing.T) {
	t.Parallel()
	now := time.Now()

	dir := t.TempDir()

	for day := 0; day <= 10; day++ {
		containerName, err := uuid.NewUUID()
		assert.NoError(t, err)

		name := now.AddDate(0, 0, -1*day).In(time.UTC).Format(timeFormat)

		err = os.MkdirAll(path.Join(dir, name, containerName.String()), 0755)
		require.NoError(t, err)
	}

	err := cleanupLogsDir(dir, 5)
	require.NoError(t, err)

	// list remaining directories
	dirList, err := os.ReadDir(dir)
	assert.NoError(t, err)
	assert.Equal(t, 5, len(dirList))
}

func TestCleanupLogsDir_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	err := cleanupLogsDir(dir, 3)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, dirList)
}

func TestCleanupLogsDir_NonExistentDir(t *testing.T) {
	t.Parallel()

	err := cleanupLogsDir("/nonexistent/path/that/does/not/exist", 3)
	require.Error(t, err)
}

func TestCleanupLogsDir_SkipsInvalidDirNames(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Now()

	name := now.AddDate(0, 0, -10).In(time.UTC).Format(timeFormat)
	require.NoError(t, os.MkdirAll(path.Join(dir, name), 0755))
	require.NoError(t, os.MkdirAll(path.Join(dir, "not-a-timestamp"), 0755))

	err := cleanupLogsDir(dir, 3)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(dirList))
	assert.Equal(t, "not-a-timestamp", dirList[0].Name())
}

func TestCleanupLogsDir_RetentionDaysOne(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Now()

	for day := 0; day < 5; day++ {
		name := now.AddDate(0, 0, -1*day).In(time.UTC).Format(timeFormat)
		require.NoError(t, os.MkdirAll(path.Join(dir, name), 0755))
	}

	err := cleanupLogsDir(dir, 1)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Equal(t, 1, len(dirList))
}

func TestScheduleLogCleanupJob_InvalidRetentionDays(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		retentionDays int
		expectErr     bool
	}{
		{name: "negative retention days", retentionDays: -1, expectErr: true},
		{name: "negative large value", retentionDays: -100, expectErr: true},
		{name: "zero uses default", retentionDays: 0, expectErr: false},
		{name: "positive value", retentionDays: 5, expectErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleaner := NewLogCleaner()
			defer cleaner.StopLogCleanupJob()

			err := cleaner.ScheduleLogCleanupJob(Config{LogsRetentionDays: tc.retentionDays})
			if tc.expectErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNewLogCleaner(t *testing.T) {
	t.Parallel()

	cleaner := NewLogCleaner()
	require.NotNil(t, cleaner)
	require.NotNil(t, cleaner.cleanerCron)
}

func TestStopLogCleanupJob_NilCron(t *testing.T) {
	t.Parallel()

	cleaner := &Cleaner{cleanerCron: nil}
	require.NotPanics(t, func() {
		cleaner.StopLogCleanupJob()
	})
}

func TestExtractTar_Directory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	header := &tar.Header{Typeflag: tar.TypeDir, Name: "subdir"}
	reader := tar.NewReader(bytes.NewReader(nil))

	err := extractTar(dir, reader, header)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "subdir"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExtractTar_RegularFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content := []byte("test log content\nline two\n")

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	header := &tar.Header{Typeflag: tar.TypeReg, Name: "test.log", Size: int64(len(content)), Mode: 0644}
	require.NoError(t, tw.WriteHeader(header))

	_, err := tw.Write(content)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)
	h, err := tr.Next()
	require.NoError(t, err)

	err = extractTar(dir, tr, h)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "test.log"))
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestExtractTar_ExistingDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "existing"), 0755))

	header := &tar.Header{Typeflag: tar.TypeDir, Name: "existing"}
	reader := tar.NewReader(bytes.NewReader(nil))

	err := extractTar(dir, reader, header)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "existing"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExtractTar_NestedDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	header := &tar.Header{Typeflag: tar.TypeDir, Name: "a/b/c"}
	reader := tar.NewReader(bytes.NewReader(nil))

	err := extractTar(dir, reader, header)
	require.NoError(t, err)

	info, err := os.Stat(filepath.Join(dir, "a/b/c"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestExtractTar_UnsupportedType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	header := &tar.Header{Typeflag: tar.TypeSymlink, Name: "link", Linkname: "target"}
	reader := tar.NewReader(bytes.NewReader(nil))

	err := extractTar(dir, reader, header)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(dir, "link"))
	assert.True(t, os.IsNotExist(err))
}

func TestExtractTar_MultipleFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	files := map[string]string{
		"log/file1.log": "content one",
		"log/file2.log": "content two",
		"log/file3.log": "content three",
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	require.NoError(t, tw.WriteHeader(&tar.Header{Typeflag: tar.TypeDir, Name: "log", Mode: 0755}))

	for name, content := range files {
		hdr := &tar.Header{Typeflag: tar.TypeReg, Name: name, Size: int64(len(content)), Mode: 0644}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())

	tr := tar.NewReader(&buf)

	for {
		h, err := tr.Next()
		if err != nil {
			break
		}

		err = extractTar(dir, tr, h)
		require.NoError(t, err)
	}

	for name, expected := range files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		require.NoError(t, err)
		assert.Equal(t, expected, string(data))
	}
}

func TestConfig_DefaultRetentionDays(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	now := time.Now()

	recent := now.AddDate(0, 0, -6).UTC().Format(timeFormat)
	old := now.AddDate(0, 0, -8).UTC().Format(timeFormat)
	require.NoError(t, os.MkdirAll(path.Join(dir, recent), 0755))
	require.NoError(t, os.MkdirAll(path.Join(dir, old), 0755))

	err := cleanupLogsDir(dir, defaultLogsRetentionDays)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirList, 1)
	assert.Equal(t, recent, dirList[0].Name())
}

func TestTimeFormat(t *testing.T) {
	t.Parallel()

	ts := time.Date(2024, 3, 15, 10, 30, 45, 0, time.UTC)
	formatted := ts.Format(timeFormat)
	assert.Equal(t, "20240315103045", formatted)

	parsed, err := time.Parse(timeFormat, formatted)
	require.NoError(t, err)
	assert.Equal(t, ts, parsed)
}

func TestCleanupLogsDir_AllExpired(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Now()

	for day := 10; day <= 15; day++ {
		name := now.AddDate(0, 0, -1*day).In(time.UTC).Format(timeFormat)
		require.NoError(t, os.MkdirAll(path.Join(dir, name), 0755))
	}

	err := cleanupLogsDir(dir, 3)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, dirList)
}

func TestCleanupLogsDir_NoneExpired(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	now := time.Now()

	for day := 0; day < 3; day++ {
		name := now.AddDate(0, 0, -1*day).In(time.UTC).Format(timeFormat)
		require.NoError(t, os.MkdirAll(path.Join(dir, name), 0755))
	}

	err := cleanupLogsDir(dir, 30)
	require.NoError(t, err)

	dirList, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Equal(t, 3, len(dirList))
}

func TestScheduleLogCleanupJob_SchedulesThenReschedules(t *testing.T) {
	t.Parallel()

	cleaner := NewLogCleaner()
	defer cleaner.StopLogCleanupJob()

	err := cleaner.ScheduleLogCleanupJob(Config{LogsRetentionDays: 5})
	require.NoError(t, err)

	entries := cleaner.cleanerCron.Entries()
	require.NotEmpty(t, entries)

	err = cleaner.ScheduleLogCleanupJob(Config{LogsRetentionDays: 10})
	require.NoError(t, err)

	assert.Len(t, cleaner.cleanerCron.Entries(), 1, "reschedule should replace old entry, not add a second one")
}

func TestScheduleLogCleanupJob_ErrorDoesNotSchedule(t *testing.T) {
	t.Parallel()

	cleaner := NewLogCleaner()
	defer cleaner.StopLogCleanupJob()

	err := cleaner.ScheduleLogCleanupJob(Config{LogsRetentionDays: -5})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value for logsRetentionDays")

	entries := cleaner.cleanerCron.Entries()
	assert.Empty(t, entries)
}
