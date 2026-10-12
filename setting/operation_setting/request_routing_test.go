package operation_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestRoutingValidation(t *testing.T) {
	threshold := 8192
	config := RequestRouting{Enabled: true, Version: "v1", StateBackend: "redis", Rules: []RoutingRule{{ID: "chat", Models: []string{"model"}, Groups: []string{"default"}, Paths: []string{"/v1/chat/completions", "/v1/messages", "/v1/responses", "/pg/chat/completions"}, Ranges: []RoutingRange{{MinTokens: 0, MaxTokens: &threshold, Instances: []RoutingInstance{{ID: "short", ChannelIDs: []int{1}, MaxInflight: 8, WorkCapacity: 100000}}}, {MinTokens: threshold, Instances: []RoutingInstance{{ID: "long", ChannelIDs: []int{2}, MaxInflight: 4, WorkCapacity: 100000}}}}}}}
	data, err := common.Marshal(config)
	require.NoError(t, err)
	parsed, err := ParseRequestRouting(string(data))
	require.NoError(t, err)
	assert.Equal(t, config, *parsed)
	config.Rules[0].Ranges[1].Instances[0].ID = "short"
	data, err = common.Marshal(config)
	require.NoError(t, err)
	_, err = ParseRequestRouting(string(data))
	require.Error(t, err, "long and short pools must not share an instance")
	config.Rules[0].Ranges[1].Instances[0].ID = "long"
	config.Rules[0].Ranges[1].Instances[0].ChannelIDs = []int{1}
	data, err = common.Marshal(config)
	require.NoError(t, err)
	_, err = ParseRequestRouting(string(data))
	require.Error(t, err, "channel aliases cannot grant additional capacity")
}
