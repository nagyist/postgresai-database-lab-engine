/*
2020 © Postgres.ai
*/

package lvm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLVMOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantLen   int
		wantFirst string
	}{
		{name: "invalid JSON", input: `not valid json`, wantErr: true},
		{name: "empty report array", input: `{"report":[]}`, wantErr: true},
		{
			name:      "valid single entry",
			input:     `{"report":[{"lv":[{"lv_name":"data","vg_name":"vg0","lv_attr":"","lv_size":"10G","pool_lv":"","origin":"","data_percent":""}]}]}`,
			wantLen:   1,
			wantFirst: "data",
		},
		{
			name:      "multiple report entries uses first only",
			input:     `{"report":[{"lv":[{"lv_name":"first","vg_name":"vg0","lv_attr":"","lv_size":"1G","pool_lv":"","origin":"","data_percent":""}]},{"lv":[{"lv_name":"second","vg_name":"vg0","lv_attr":"","lv_size":"2G","pool_lv":"","origin":"","data_percent":""}]}]}`,
			wantLen:   1,
			wantFirst: "first",
		},
		{name: "empty volumes in report", input: `{"report":[{"lv":[]}]}`, wantLen: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumes, err := parseLVMOutput(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, volumes, tt.wantLen)

			if tt.wantFirst != "" {
				require.NotEmpty(t, volumes)
				assert.Equal(t, tt.wantFirst, volumes[0].Name)
			}
		})
	}
}
