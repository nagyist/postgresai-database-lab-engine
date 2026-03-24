/*
2020 © Postgres.ai
*/

package physical

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitParamsExtraction(t *testing.T) {
	controlDataOutput := bytes.NewBufferString(`
wal_level setting:                    logical
wal_log_hints setting:                on
max_connections setting:              500
max_worker_processes setting:         8
max_prepared_xacts setting:           3
max_locks_per_xact setting:           128
track_commit_timestamp setting:       off
max_wal_senders setting:              15
`)

	expectedSettings := map[string]string{
		"max_connections":           "500",
		"max_locks_per_transaction": "128",
		"max_prepared_transactions": "3",
		"max_worker_processes":      "8",
		"track_commit_timestamp":    "off",
		"max_wal_senders":           "15",
	}

	settings, err := extractControlDataParams(context.Background(), controlDataOutput)

	require.Nil(t, err)
	assert.EqualValues(t, settings, expectedSettings)
}

func TestInitParamsExtraction_EmptyInput(t *testing.T) {
	settings, err := extractControlDataParams(context.Background(), bytes.NewBufferString(""))
	require.NoError(t, err)
	assert.Empty(t, settings)
}

func TestInitParamsExtraction_PartialData(t *testing.T) {
	controlDataOutput := bytes.NewBufferString(`
wal_level setting:                    replica
max_connections setting:              100
`)

	settings, err := extractControlDataParams(context.Background(), controlDataOutput)
	require.NoError(t, err)
	assert.Equal(t, "100", settings["max_connections"])
	assert.Len(t, settings, 1)
}

func TestInitParamsExtraction_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	controlDataOutput := bytes.NewBufferString(`
max_connections setting:              500
max_worker_processes setting:         8
`)

	settings, err := extractControlDataParams(ctx, controlDataOutput)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, settings)
}

func TestInitParamsExtraction_NoMatchingParams(t *testing.T) {
	controlDataOutput := bytes.NewBufferString(`
wal_level setting:                    logical
wal_log_hints setting:                on
some_other_param setting:             value
`)

	settings, err := extractControlDataParams(context.Background(), controlDataOutput)
	require.NoError(t, err)
	assert.Empty(t, settings)
}
