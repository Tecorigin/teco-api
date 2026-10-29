package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupTokenRatios(t *testing.T) {
	inputBefore := GroupInputRatio2JSONString()
	outputBefore := GroupOutputRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, UpdateGroupInputRatioByJSONString(inputBefore))
		require.NoError(t, UpdateGroupOutputRatioByJSONString(outputBefore))
	})
	require.NoError(t, UpdateGroupInputRatioByJSONString(`{"vip":0.5,"free":0}`))
	require.NoError(t, UpdateGroupOutputRatioByJSONString(`{"vip":0.8,"free":0}`))
	for _, test := range []struct {
		name          string
		input, output float64
	}{
		{"vip", 0.5, 0.8},
		{"free", 0, 0},
		{"missing", 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, output := GetGroupTokenRatios(test.name)
			assert.Equal(t, test.input, input)
			assert.Equal(t, test.output, output)
		})
	}
	for _, value := range []string{`{"vip":-1}`, `null`, `[]`, `{"vip":1e309}`, `{"":1}`} {
		assert.Error(t, UpdateGroupInputRatioByJSONString(value))
		assert.Error(t, UpdateGroupOutputRatioByJSONString(value))
	}
	input, output := GetGroupTokenRatios("vip")
	assert.Equal(t, 0.5, input)
	assert.Equal(t, 0.8, output)
}
