//go:build linux && !s390x && !arm && !386
// +build linux,!s390x,!arm,!386

package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/postgres-ai/database-lab/v3/internal/provision/thinclones/zfs"
)

func TestDetectFSType(t *testing.T) {
	tests := []struct {
		name     string
		fsType   int64
		expected string
	}{
		{name: "ext4 filesystem", fsType: 0xef53, expected: ext4},
		{name: "zfs filesystem", fsType: 0x2fc12fc1, expected: zfs.PoolMode},
		{name: "unknown filesystem returns empty string", fsType: 0x1234, expected: ""},
		{name: "zero value returns empty string", fsType: 0, expected: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, detectFSType(tc.fsType))
		})
	}
}
