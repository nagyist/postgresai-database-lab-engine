package branching

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchName(t *testing.T) {
	testCases := []struct {
		poolName string
		branch   string
		expected string
	}{
		{poolName: "pool", branch: "main", expected: "pool/branch/main"},
		{poolName: "pool/pg17", branch: "dev", expected: "pool/pg17/branch/dev"},
		{poolName: "pool", branch: "feature/test", expected: "pool/branch/feature/test"},
	}

	for _, tc := range testCases {
		t.Run(tc.poolName+"/"+tc.branch, func(t *testing.T) {
			result := BranchName(tc.poolName, tc.branch)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCloneDataset(t *testing.T) {
	testCases := []struct {
		poolName  string
		branch    string
		cloneName string
		expected  string
	}{
		{poolName: "pool", branch: "main", cloneName: "clone001", expected: "pool/branch/main/clone001"},
		{poolName: "pool/pg17", branch: "dev", cloneName: "abc123", expected: "pool/pg17/branch/dev/abc123"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := CloneDataset(tc.poolName, tc.branch, tc.cloneName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCloneName(t *testing.T) {
	testCases := []struct {
		poolName  string
		branch    string
		cloneName string
		revision  int
		expected  string
	}{
		{poolName: "pool", branch: "main", cloneName: "clone001", revision: 0, expected: "pool/branch/main/clone001/r0"},
		{poolName: "pool/pg17", branch: "dev", cloneName: "abc", revision: 5, expected: "pool/pg17/branch/dev/abc/r5"},
		{poolName: "pool", branch: "main", cloneName: "c1", revision: 123, expected: "pool/branch/main/c1/r123"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := CloneName(tc.poolName, tc.branch, tc.cloneName, tc.revision)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRevisionSegment(t *testing.T) {
	testCases := []struct {
		revision int
		expected string
	}{
		{revision: 0, expected: "r0"},
		{revision: 1, expected: "r1"},
		{revision: 42, expected: "r42"},
		{revision: 999, expected: "r999"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := RevisionSegment(tc.revision)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseCloneName(t *testing.T) {
	const poolName = "pool/pg17"

	testCases := []struct {
		name         string
		cloneDataset string
		expectedName string
		expectedOk   bool
	}{
		{name: "valid clone dataset", cloneDataset: "pool/pg17/branch/main/clone001/r0", expectedName: "clone001", expectedOk: true},
		{name: "missing clone segment", cloneDataset: "pool/pg17/branch/main", expectedName: "", expectedOk: false},
		{name: "empty input", cloneDataset: "", expectedName: "", expectedOk: false},
		{name: "wrong pool prefix", cloneDataset: "other/branch/main/clone001/r0", expectedName: "branch", expectedOk: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			name, ok := ParseCloneName(tc.cloneDataset, poolName)
			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedName, name)
		})
	}
}

func TestParseBranchName(t *testing.T) {
	const poolName = "pool/pg17"

	testCases := []struct {
		name           string
		cloneDataset   string
		expectedBranch string
		expectedOk     bool
	}{
		{name: "valid clone dataset", cloneDataset: "pool/pg17/branch/main/clone001/r0", expectedBranch: "main", expectedOk: true},
		{name: "dev branch", cloneDataset: "pool/pg17/branch/dev/abc123/r5", expectedBranch: "dev", expectedOk: true},
		{name: "missing segments", cloneDataset: "pool/pg17/branch/main", expectedBranch: "", expectedOk: false},
		{name: "empty input", cloneDataset: "", expectedBranch: "", expectedOk: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			branch, ok := ParseBranchName(tc.cloneDataset, poolName)
			assert.Equal(t, tc.expectedOk, ok)
			assert.Equal(t, tc.expectedBranch, branch)
		})
	}
}

func TestParseBaseDatasetFromSnapshot(t *testing.T) {
	testCases := []struct {
		name     string
		snapshot string
		expected string
	}{
		{name: "snapshot with branch path", snapshot: "pool/pg17/branch/main/clone001/r0@snap1", expected: "pool/pg17"},
		{name: "snapshot without branch dir", snapshot: "pool/pg17@snapshot_20250407", expected: "pool/pg17"},
		{name: "no @ separator", snapshot: "pool/pg17/branch/main", expected: ""},
		{name: "empty input", snapshot: "", expected: ""},
		{name: "deep pool with branch", snapshot: "a/b/c/branch/dev/clone/r0@snap", expected: "a/b/c"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ParseBaseDatasetFromSnapshot(tc.snapshot)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParsingBranchNameFromSnapshot(t *testing.T) {
	const poolName = "pool/pg17"

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "pool/pg17@snapshot_20250407101616",
			expected: "",
		},
		{
			input:    "pool/pg17/branch/dev@20250407101828",
			expected: "dev",
		},
		{
			input:    "pool/pg17/branch/main/cvpqe8gn9i6s73b49e3g/r0@20250407102140",
			expected: "main",
		},
	}

	for _, tc := range testCases {
		branchName := ParseBranchNameFromSnapshot(tc.input, poolName)

		assert.Equal(t, tc.expected, branchName)
	}
}
