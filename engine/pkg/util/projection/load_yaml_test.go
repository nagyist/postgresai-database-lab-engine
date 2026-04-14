package projection

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadYaml(t *testing.T) {
	r := require.New(t)
	s := &testStruct{}
	node := getYamlNormal(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireMissEmpty(t, s)
	requireComplete(t, s)
}

func TestLoadYamlNull(t *testing.T) {
	r := require.New(t)
	s := fullTestStruct()
	node := getYamlNull(t)

	err := LoadYaml(s, node, LoadOptions{})
	r.NoError(err)

	requireEmpty(t, s)
}

func TestLoadYaml_MergeKey(t *testing.T) {
	type mergeStruct struct {
		Configs      map[string]interface{} `proj:"parent.options.configs"`
		ParallelJobs *int64                 `proj:"parent.options.parallelJobs"`
	}

	const yamlData = `
defaults: &defaults
  configs:
    shared_buffers: 1GB
    work_mem: 100MB

parent:
  options:
    <<: *defaults
    parallelJobs: 4
`

	node := &yaml.Node{}
	err := yaml.Unmarshal([]byte(yamlData), node)
	require.NoError(t, err)

	s := &mergeStruct{}
	err = LoadYaml(s, node, LoadOptions{})
	require.NoError(t, err)

	require.Equal(t, map[string]interface{}{"shared_buffers": "1GB", "work_mem": "100MB"}, s.Configs)
	require.Equal(t, int64(4), *s.ParallelJobs)
}
