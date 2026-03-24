package pglog

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPostgresLastActivity(t *testing.T) {
	testCases := []struct {
		logTime      string
		logMessage   string
		timeActivity *time.Time
		loc          *time.Location
	}{
		{
			logTime:      "2020-01-10 11:49:14.615 UTC",
			logMessage:   "duration: 9.893 ms  statement: SELECT 1;",
			loc:          time.UTC,
			timeActivity: pointer.ToTime(time.Date(2020, 1, 10, 11, 49, 14, 615000000, time.UTC)),
		},
		{
			logTime:      "2020-01-10 11:49:14.615 CET",
			logMessage:   "duration: 9.893 ms  statement: SELECT 1;",
			loc:          time.FixedZone("CET", 3600),
			timeActivity: pointer.ToTime(time.Date(2020, 1, 10, 11, 49, 14, 615000000, time.FixedZone("CET", 3600))),
		},
		{
			logTime:      "2020-01-11 13:10:58.503 UTC",
			logMessage:   "duration: 0.077 ms  statement:",
			loc:          time.UTC,
			timeActivity: pointer.ToTime(time.Date(2020, 1, 11, 13, 10, 58, 503000000, time.UTC)),
		},
		{
			logTime:      "2020-01-11 12:10:56.867 UTC",
			logMessage:   "database system is ready to accept connections",
			timeActivity: nil,
		},
		{
			logTime:      "",
			logMessage:   "duration: 9.893 ms  statement: SELECT 1;",
			timeActivity: nil,
		},
		{
			logTime:      "2021-03-24 15:33:56.135 UTC",
			logMessage:   "duration: 0.544 ms  execute lrupsc_28_0: EXPLAIN (FORMAT TEXT) select 1",
			timeActivity: pointer.ToTime(time.Date(2021, 3, 24, 15, 33, 56, 135000000, time.UTC)),
		},
	}

	for _, tc := range testCases {
		lastActivity, err := ParsePostgresLastActivity(tc.logTime, tc.logMessage, tc.loc)
		require.NoError(t, err)
		assert.Equal(t, tc.timeActivity, lastActivity)
	}
}

func TestGetPostgresLastActivityWhenFailedParseTime(t *testing.T) {
	testCases := []struct {
		logTime     string
		logMessage  string
		errorString string
	}{
		{
			logTime:     "2020-01-10 11:49:14",
			logMessage:  "duration: 9.893 ms  statement: SELECT 1;",
			errorString: `failed to parse the last activity time: parsing time "2020-01-10 11:49:14" as "2006-01-02 15:04:05.000 MST": cannot parse "" as ".000"`,
		},
	}

	for _, tc := range testCases {
		lastActivity, err := ParsePostgresLastActivity(tc.logTime, tc.logMessage, time.UTC)
		require.Nil(t, lastActivity)
		assert.EqualError(t, err, tc.errorString)
	}
}

func createLogDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	logDir := path.Join(dir, "log")
	require.NoError(t, os.Mkdir(logDir, 0o755))

	return dir
}

func createFile(t *testing.T, dir, name string) {
	t.Helper()

	f, err := os.Create(path.Join(dir, "log", name))
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestSelector_DiscoverLogDir(t *testing.T) {
	dir := createLogDir(t)
	createFile(t, dir, "postgresql-2020-01-10_114914.csv")
	createFile(t, dir, "postgresql-2020-01-11_131058.csv")
	createFile(t, dir, "postgresql-2020-01-09_080000.csv")

	s := NewSelector(dir)
	require.NoError(t, s.DiscoverLogDir())

	assert.Equal(t, []string{
		"postgresql-2020-01-09_080000.csv",
		"postgresql-2020-01-10_114914.csv",
		"postgresql-2020-01-11_131058.csv",
	}, s.fileNames, "files should be sorted alphabetically")
}

func TestSelector_DiscoverLogDir_EmptyDir(t *testing.T) {
	dir := createLogDir(t)

	s := NewSelector(dir)
	err := s.DiscoverLogDir()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "log files not found")
}

func TestSelector_DiscoverLogDir_SkipsNonCSV(t *testing.T) {
	dir := createLogDir(t)
	createFile(t, dir, "postgresql-2020-01-10_114914.csv")
	createFile(t, dir, "postgresql-2020-01-10_114914.log")
	createFile(t, dir, "some-other-file.txt")

	s := NewSelector(dir)
	require.NoError(t, s.DiscoverLogDir())

	assert.Equal(t, []string{"postgresql-2020-01-10_114914.csv"}, s.fileNames, "only csv files should be discovered")
}

func TestSelector_DiscoverLogDir_NonexistentDir(t *testing.T) {
	s := NewSelector("/nonexistent/path")
	err := s.DiscoverLogDir()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read a log directory")
}

func TestSelector_DiscoverLogDir_SkipsSubdirectories(t *testing.T) {
	dir := createLogDir(t)
	createFile(t, dir, "postgresql-2020-01-10_114914.csv")
	require.NoError(t, os.Mkdir(path.Join(dir, "log", "subdir"), 0o755))

	s := NewSelector(dir)
	require.NoError(t, s.DiscoverLogDir())

	assert.Equal(t, []string{"postgresql-2020-01-10_114914.csv"}, s.fileNames)
}

func TestSelector_Next(t *testing.T) {
	dir := createLogDir(t)
	createFile(t, dir, "postgresql-2020-01-09_080000.csv")
	createFile(t, dir, "postgresql-2020-01-10_114914.csv")

	s := NewSelector(dir)
	require.NoError(t, s.DiscoverLogDir())

	logDir := path.Join(dir, "log")

	first, err := s.Next()
	require.NoError(t, err)
	assert.Equal(t, path.Join(logDir, "postgresql-2020-01-09_080000.csv"), first)

	second, err := s.Next()
	require.NoError(t, err)
	assert.Equal(t, path.Join(logDir, "postgresql-2020-01-10_114914.csv"), second)

	_, err = s.Next()
	assert.ErrorIs(t, err, ErrLastFile, "should return ErrLastFile after all files consumed")
}

func TestSelector_Next_EmptyFileNames(t *testing.T) {
	s := NewSelector(t.TempDir())

	result, err := s.Next()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "log fileNames not found")
	assert.Empty(t, result)
}

func TestSelector_FilterOldFilesInList(t *testing.T) {
	testCases := []struct {
		name        string
		files       []string
		minimumTime time.Time
		expected    []string
	}{
		{
			name:        "filters old files keeping the file before minimum time",
			files:       []string{"postgresql-2020-01-08_080000.csv", "postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv", "postgresql-2020-01-11_131058.csv"},
			minimumTime: time.Date(2020, 1, 10, 12, 0, 0, 0, time.UTC),
			expected:    []string{"postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv", "postgresql-2020-01-11_131058.csv"},
		},
		{
			name:        "zero minimum time keeps all files",
			files:       []string{"postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv"},
			minimumTime: time.Time{},
			expected:    []string{"postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv"},
		},
		{
			name:        "minimum time before all files keeps all",
			files:       []string{"postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv"},
			minimumTime: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC),
			expected:    []string{"postgresql-2020-01-09_080000.csv", "postgresql-2020-01-10_114914.csv"},
		},
		{
			name:        "single file is preserved",
			files:       []string{"postgresql-2020-01-10_114914.csv"},
			minimumTime: time.Date(2020, 1, 11, 0, 0, 0, 0, time.UTC),
			expected:    []string{"postgresql-2020-01-10_114914.csv"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Selector{fileNames: tc.files}
			s.SetMinimumTime(tc.minimumTime)
			s.FilterOldFilesInList()
			assert.Equal(t, tc.expected, s.fileNames)
		})
	}
}

func TestSelector_SetMinimumTime(t *testing.T) {
	s := NewSelector("/tmp")
	minTime := time.Date(2020, 6, 15, 10, 30, 0, 0, time.UTC)

	s.SetMinimumTime(minTime)

	assert.Equal(t, minTime, s.minimumTime)
}

func TestBuildLogDirName(t *testing.T) {
	assert.Equal(t, "/data/clone/log", buildLogDirName("/data/clone"))
	assert.Equal(t, "mydir/log", buildLogDirName("mydir"))
}

func TestNewSelector(t *testing.T) {
	s := NewSelector("/data/clone")

	assert.Equal(t, "/data/clone/log", s.logDir)
	assert.Empty(t, s.fileNames)
	assert.Equal(t, 0, s.currentIndex)
}
