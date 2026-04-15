/*
2026 © PostgresAI
*/

package rdsrefresh

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseInstanceClass(t *testing.T) {
	testCases := []struct {
		instanceClass  string
		expectedFamily string
		expectedSize   string
	}{
		{instanceClass: "db.m5.xlarge", expectedFamily: "m5", expectedSize: "xlarge"},
		{instanceClass: "db.t3.medium", expectedFamily: "t3", expectedSize: "medium"},
		{instanceClass: "db.r6g.2xlarge", expectedFamily: "r6g", expectedSize: "2xlarge"},
		{instanceClass: "db.m5.metal", expectedFamily: "m5", expectedSize: "metal"},
		{instanceClass: "db.t3.micro", expectedFamily: "t3", expectedSize: "micro"},
		{instanceClass: "db.r6gd.16xlarge", expectedFamily: "r6gd", expectedSize: "16xlarge"},
	}

	for _, tc := range testCases {
		t.Run(tc.instanceClass, func(t *testing.T) {
			family, size, err := parseInstanceClass(tc.instanceClass)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedFamily, family)
			assert.Equal(t, tc.expectedSize, size)
		})
	}
}

func TestParseInstanceClassErrors(t *testing.T) {
	testCases := []struct {
		name          string
		instanceClass string
	}{
		{name: "missing db prefix", instanceClass: "m5.xlarge"},
		{name: "missing size", instanceClass: "db.m5"},
		{name: "trailing dot only", instanceClass: "db."},
		{name: "empty string", instanceClass: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseInstanceClass(tc.instanceClass)
			require.Error(t, err)
		})
	}
}

func TestResolveRDSInstanceVCPUs(t *testing.T) {
	testCases := []struct {
		instanceClass string
		expectedVCPUs int
	}{
		{instanceClass: "db.t3.micro", expectedVCPUs: 2},
		{instanceClass: "db.t3.small", expectedVCPUs: 2},
		{instanceClass: "db.t3.medium", expectedVCPUs: 2},
		{instanceClass: "db.m5.large", expectedVCPUs: 2},
		{instanceClass: "db.m5.xlarge", expectedVCPUs: 4},
		{instanceClass: "db.r6g.2xlarge", expectedVCPUs: 8},
		{instanceClass: "db.r6g.4xlarge", expectedVCPUs: 16},
		{instanceClass: "db.r6g.8xlarge", expectedVCPUs: 32},
		{instanceClass: "db.r6g.16xlarge", expectedVCPUs: 64},
		{instanceClass: "db.m5.24xlarge", expectedVCPUs: 96},
		{instanceClass: "db.m5.5xlarge", expectedVCPUs: 20},
		{instanceClass: "db.m5.metal", expectedVCPUs: 96},
		{instanceClass: "db.m6g.metal", expectedVCPUs: 64},
		{instanceClass: "db.m6i.metal", expectedVCPUs: 128},
		{instanceClass: "db.m5d.metal", expectedVCPUs: 96},
		{instanceClass: "db.r6gd.metal", expectedVCPUs: 64},
	}

	for _, tc := range testCases {
		t.Run(tc.instanceClass, func(t *testing.T) {
			vcpus, err := resolveRDSInstanceVCPUs(tc.instanceClass)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedVCPUs, vcpus)
		})
	}
}

func TestResolveRDSInstanceVCPUsErrors(t *testing.T) {
	testCases := []struct {
		name          string
		instanceClass string
	}{
		{name: "invalid format", instanceClass: "invalid"},
		{name: "missing size", instanceClass: "db.m5"},
		{name: "unknown size", instanceClass: "db.m5.unknown"},
		{name: "unknown metal family", instanceClass: "db.z99.metal"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveRDSInstanceVCPUs(tc.instanceClass)
			require.Error(t, err)
		})
	}
}

func TestParseXlargeMultiplier(t *testing.T) {
	testCases := []struct {
		size          string
		expectedVCPUs int
	}{
		{size: "2xlarge", expectedVCPUs: 8},
		{size: "4xlarge", expectedVCPUs: 16},
		{size: "5xlarge", expectedVCPUs: 20},
	}

	for _, tc := range testCases {
		t.Run(tc.size, func(t *testing.T) {
			vcpus, err := parseXlargeMultiplier(tc.size)

			require.NoError(t, err)
			assert.Equal(t, tc.expectedVCPUs, vcpus)
		})
	}
}

func TestParseXlargeMultiplierErrors(t *testing.T) {
	testCases := []struct {
		name string
		size string
	}{
		{name: "no multiplier prefix", size: "xlarge"},
		{name: "not xlarge variant", size: "large"},
		{name: "non-numeric prefix", size: "abcxlarge"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseXlargeMultiplier(tc.size)
			require.Error(t, err)
		})
	}
}

func TestResolveLocalVCPUs(t *testing.T) {
	vcpus := resolveLocalVCPUs()

	assert.Equal(t, runtime.NumCPU(), vcpus)
	assert.GreaterOrEqual(t, vcpus, minParallelJobs)
}

func TestResolveParallelism(t *testing.T) {
	t.Run("resolves both dump and restore jobs", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{InstanceClass: "db.m5.xlarge"}}

		result, err := ResolveParallelism(cfg)

		require.NoError(t, err)
		assert.Equal(t, 2, result.DumpJobs)
		assert.Equal(t, runtime.NumCPU(), result.RestoreJobs)
	})

	t.Run("returns error for invalid instance class", func(t *testing.T) {
		cfg := &Config{RDSClone: RDSCloneConfig{InstanceClass: "invalid"}}

		_, err := ResolveParallelism(cfg)

		require.Error(t, err)
	})
}
