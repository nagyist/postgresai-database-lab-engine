package retrieval

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/postgres-ai/database-lab/v3/internal/retrieval/config"
)

func TestParallelJobSpecs(t *testing.T) {
	testCases := []config.Config{
		{
			Jobs: []string{"logicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"logicalRestore": {},
			},
		},
		{
			Jobs: []string{"physicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"physicalRestore": {},
			},
		},
		{
			Jobs: []string{"logicalDump"},
			JobsSpec: map[string]config.JobSpec{
				"logicalDump": {},
			},
		},
		{
			Jobs: []string{"logicalDump", "logicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"logicalDump":    {},
				"logicalRestore": {},
			},
		},
	}

	for _, tc := range testCases {
		err := validateStructure(&tc)
		assert.Nil(t, err)
	}

}

func TestInvalidParallelJobSpecs(t *testing.T) {
	testCases := []config.Config{
		{
			Jobs: []string{"logicalRestore", "physicalRestore"},
			JobsSpec: map[string]config.JobSpec{
				"physicalRestore": {},
				"logicalRestore":  {},
			},
		},
	}

	for _, tc := range testCases {
		err := validateStructure(&tc)
		assert.Error(t, err)
	}
}

func TestPhysicalJobs(t *testing.T) {
	testCases := []struct {
		spec        map[string]config.JobSpec
		hasPhysical bool
	}{
		{
			spec:        map[string]config.JobSpec{"physicalSnapshot": {}},
			hasPhysical: true,
		},
		{
			spec:        map[string]config.JobSpec{"physicalRestore": {}},
			hasPhysical: true,
		},
		{
			spec: map[string]config.JobSpec{
				"physicalSnapshot": {},
				"physicalRestore":  {},
			},
			hasPhysical: true,
		},
		{
			spec:        map[string]config.JobSpec{},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalDump": {}},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalRestore": {}},
			hasPhysical: false,
		},
		{
			spec:        map[string]config.JobSpec{"logicalSnapshot": {}},
			hasPhysical: false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.hasPhysical, hasPhysicalJob(tc.spec))
	}
}

func TestLogicalJobs(t *testing.T) {
	testCases := []struct {
		spec       map[string]config.JobSpec
		hasLogical bool
	}{
		{
			spec:       map[string]config.JobSpec{"logicalSnapshot": {}},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{"logicalRestore": {}},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{"logicalDump": {}},
			hasLogical: true,
		},
		{
			spec: map[string]config.JobSpec{
				"logicalDump":     {},
				"logicalRestore":  {},
				"logicalSnapshot": {},
			},
			hasLogical: true,
		},
		{
			spec:       map[string]config.JobSpec{},
			hasLogical: false,
		},
		{
			spec:       map[string]config.JobSpec{"physicalRestore": {}},
			hasLogical: false,
		},
		{
			spec:       map[string]config.JobSpec{"physicalSnapshot": {}},
			hasLogical: false,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.hasLogical, hasLogicalJob(tc.spec))
	}
}

func TestFormatJobsSpec(t *testing.T) {
	t.Run("enriches job specs with names", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"logicalDump", "logicalRestore"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {Options: map[string]interface{}{"key": "val"}}, "logicalRestore": {}},
		}

		result, err := formatJobsSpec(cfg)
		require.NoError(t, err)
		assert.Equal(t, "logicalDump", result.JobsSpec["logicalDump"].Name)
		assert.Equal(t, "logicalRestore", result.JobsSpec["logicalRestore"].Name)
		assert.Equal(t, "val", result.JobsSpec["logicalDump"].Options["key"])
	})

	t.Run("returns error for undefined jobs", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"logicalDump", "missingJob"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}},
		}

		result, err := formatJobsSpec(cfg)
		assert.Nil(t, result)
		assert.ErrorContains(t, err, "missingJob")
		assert.ErrorContains(t, err, "config contains jobs without specification")
	})

	t.Run("returns error for multiple undefined jobs", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"job1", "job2"},
			JobsSpec: map[string]config.JobSpec{},
		}

		_, err := formatJobsSpec(cfg)
		assert.ErrorContains(t, err, "job1")
		assert.ErrorContains(t, err, "job2")
	})

	t.Run("handles empty jobs list", func(t *testing.T) {
		cfg := &config.Config{Jobs: []string{}, JobsSpec: map[string]config.JobSpec{}}

		result, err := formatJobsSpec(cfg)
		require.NoError(t, err)
		assert.Empty(t, result.JobsSpec)
	})

	t.Run("preserves refresh config", func(t *testing.T) {
		refresh := &config.Refresh{Timetable: "0 * * * *"}
		cfg := &config.Config{Refresh: refresh, Jobs: []string{}, JobsSpec: map[string]config.JobSpec{}}

		result, err := formatJobsSpec(cfg)
		require.NoError(t, err)
		assert.Equal(t, refresh, result.Refresh)
	})

	t.Run("only includes jobs from the jobs list", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"logicalDump"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}, "logicalRestore": {}},
		}

		result, err := formatJobsSpec(cfg)
		require.NoError(t, err)
		assert.Len(t, result.JobsSpec, 1)
		assert.Contains(t, result.JobsSpec, "logicalDump")
	})
}

func TestValidateRefreshTimetable(t *testing.T) {
	t.Run("nil refresh is valid", func(t *testing.T) {
		cfg := &config.Config{}
		assert.NoError(t, validateRefreshTimetable(cfg))
	})

	t.Run("empty timetable is valid", func(t *testing.T) {
		cfg := &config.Config{Refresh: &config.Refresh{Timetable: ""}}
		assert.NoError(t, validateRefreshTimetable(cfg))
	})

	t.Run("valid cron expression", func(t *testing.T) {
		cfg := &config.Config{Refresh: &config.Refresh{Timetable: "0 */6 * * *"}}
		assert.NoError(t, validateRefreshTimetable(cfg))
	})

	t.Run("invalid cron expression", func(t *testing.T) {
		cfg := &config.Config{Refresh: &config.Refresh{Timetable: "invalid-cron"}}
		err := validateRefreshTimetable(cfg)
		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid timetable")
	})

	t.Run("valid minute-level cron expression", func(t *testing.T) {
		cfg := &config.Config{Refresh: &config.Refresh{Timetable: "30 2 * * 1"}}
		assert.NoError(t, validateRefreshTimetable(cfg))
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid logical config", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"logicalDump", "logicalRestore", "logicalSnapshot"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}, "logicalRestore": {}, "logicalSnapshot": {}},
		}

		result, err := ValidateConfig(cfg)
		require.NoError(t, err)
		assert.Equal(t, "logicalDump", result.JobsSpec["logicalDump"].Name)
	})

	t.Run("valid physical config with timetable", func(t *testing.T) {
		cfg := &config.Config{
			Refresh:  &config.Refresh{Timetable: "0 3 * * *"},
			Jobs:     []string{"physicalRestore", "physicalSnapshot"},
			JobsSpec: map[string]config.JobSpec{"physicalRestore": {}, "physicalSnapshot": {}},
		}

		result, err := ValidateConfig(cfg)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("fails on undefined job", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"nonexistent"},
			JobsSpec: map[string]config.JobSpec{},
		}

		_, err := ValidateConfig(cfg)
		assert.ErrorContains(t, err, "nonexistent")
	})

	t.Run("fails on invalid timetable", func(t *testing.T) {
		cfg := &config.Config{
			Refresh:  &config.Refresh{Timetable: "bad"},
			Jobs:     []string{"logicalDump"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}},
		}

		_, err := ValidateConfig(cfg)
		assert.ErrorContains(t, err, "invalid timetable")
	})

	t.Run("fails on mixed logical and physical jobs", func(t *testing.T) {
		cfg := &config.Config{
			Jobs:     []string{"logicalDump", "physicalRestore"},
			JobsSpec: map[string]config.JobSpec{"logicalDump": {}, "physicalRestore": {}},
		}

		_, err := ValidateConfig(cfg)
		assert.ErrorContains(t, err, "must not contain physical and logical jobs simultaneously")
	})

	t.Run("empty config is valid", func(t *testing.T) {
		cfg := &config.Config{Jobs: []string{}, JobsSpec: map[string]config.JobSpec{}}

		result, err := ValidateConfig(cfg)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}
