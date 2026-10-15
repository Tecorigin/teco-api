package service

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRoutingFeaturesProtocols(t *testing.T) {
	for _, sample := range []struct {
		path string
		body string
	}{
		{"/v1/chat/completions", `{"messages":[{"role":"user","content":"hello"}],"max_tokens":20}`},
		{"/v1/messages", `{"system":"system","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}],"max_tokens":20}`},
		{"/v1/responses", `{"instructions":"system","input":"hello","max_output_tokens":20}`},
	} {
		tokens, work, err := routingFeatures([]byte(sample.body), "routing-model", sample.path)
		require.NoError(t, err)
		assert.Greater(t, tokens, 0)
		assert.Less(t, tokens, 256)
		assert.Equal(t, float64(tokens), work)
	}
	for _, body := range []string{`{"input":"hello","background":true}`, `{"input":"hello","previous_response_id":"resp_1"}`, `{"messages":[{"content":[{"type":"image_url","image_url":{"url":"https://example.com/image"}}]}]}`} {
		_, _, err := routingFeatures([]byte(body), "routing-model", "/v1/responses")
		require.Error(t, err)
	}
	tokens, work, err := routingFeatures([]byte(`{"input":"hello","max_output_tokens":2}`), "routing-model", "/v1/responses")
	require.NoError(t, err)
	assert.Equal(t, float64(tokens), work)
	for _, body := range []string{`{"input":"hello"}`, `{"input":"hello","max_output_tokens":2000}`} {
		otherTokens, otherWork, err := routingFeatures([]byte(body), "routing-model", "/v1/responses")
		require.NoError(t, err)
		assert.Equal(t, tokens, otherTokens)
		assert.Equal(t, work, otherWork)
	}
	assert.Equal(t, CountTextToken("hello", "routing-model")+3, tokens)
}

func TestRequestRoutingTokenRangeFallback(t *testing.T) {
	previousDB, previousCache := model.DB, common.MemoryCacheEnabled
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, common.MemoryCacheEnabled = database, true
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled = previousDB, previousCache
		if previousDB != nil && previousCache {
			model.InitChannelCache()
		}
	})
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Ability{}))
	require.NoError(t, database.Create(&model.Channel{Id: 101, Type: 1, Status: common.ChannelStatusEnabled, Models: "routing-model", Group: "default"}).Error)
	require.NoError(t, database.Create(&model.Ability{ChannelId: 101, Model: "routing-model", Group: "default", Enabled: true}).Error)
	model.InitChannelCache()
	common.OptionMapRWMutex.Lock()
	previousOptionMap := common.OptionMap
	previousOption, existed := common.OptionMap[operation_setting.RequestRoutingOption]
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		defer common.OptionMapRWMutex.Unlock()
		if previousOptionMap == nil {
			common.OptionMap = nil
		} else if existed {
			common.OptionMap[operation_setting.RequestRoutingOption] = previousOption
		} else {
			delete(common.OptionMap, operation_setting.RequestRoutingOption)
		}
	})
	for _, sample := range []struct {
		path string
		body string
	}{
		{"/v1/chat/completions", `{"messages":[{"role":"user","content":"hello"}]}`},
		{"/pg/chat/completions", `{"messages":[{"role":"user","content":"hello"}]}`},
		{"/v1/messages", `{"messages":[{"role":"user","content":"hello"}]}`},
		{"/v1/responses", `{"input":"hello"}`},
	} {
		t.Run(sample.path, func(t *testing.T) {
			tokens, _, err := routingFeatures([]byte(sample.body), "routing-model", sample.path)
			require.NoError(t, err)
			upper := tokens + 1
			for _, scenario := range []struct {
				name   string
				ranges []operation_setting.RoutingRange
				active bool
			}{
				{"below minimum", []operation_setting.RoutingRange{{MinTokens: upper}}, false},
				{"gap", []operation_setting.RoutingRange{{MaxTokens: &tokens}, {MinTokens: upper}}, false},
				{"exclusive maximum", []operation_setting.RoutingRange{{MaxTokens: &tokens}}, false},
				{"inclusive minimum", []operation_setting.RoutingRange{{MinTokens: tokens, MaxTokens: &upper}}, true},
				{"inside range", []operation_setting.RoutingRange{{MaxTokens: &upper}}, true},
				{"unbounded range", []operation_setting.RoutingRange{{MinTokens: tokens}}, true},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					for index := range scenario.ranges {
						scenario.ranges[index].Instances = []operation_setting.RoutingInstance{{ID: fmt.Sprintf("instance-%d", index), ChannelIDs: []int{102 + index}}}
					}
					config := operation_setting.RequestRouting{Enabled: true, Version: "test", Rules: []operation_setting.RoutingRule{{ID: "test", Models: []string{"routing-model"}, Ranges: scenario.ranges}}}
					raw, err := common.Marshal(config)
					require.NoError(t, err)
					common.OptionMapRWMutex.Lock()
					common.OptionMap[operation_setting.RequestRoutingOption] = string(raw)
					common.OptionMapRWMutex.Unlock()
					recorder := httptest.NewRecorder()
					ctx, _ := gin.CreateTestContext(recorder)
					ctx.Request = httptest.NewRequest("POST", sample.path, strings.NewReader(sample.body))
					require.NoError(t, PrepareRequestRouting(ctx, "routing-model", "default"))
					require.NoError(t, PrepareRequestRouting(ctx, "routing-model", "default"))
					assert.Equal(t, scenario.active, RequestRoutingActive(ctx))
					if scenario.active {
						assert.Equal(t, "test", recorder.Header().Get("X-Teco-Routing-Version"))
						assert.NotEmpty(t, recorder.Header().Get("X-Teco-Routing-Class"))
						return
					}
					for _, header := range []string{"X-Teco-Routing-Version", "X-Teco-Routing-Class", "X-Teco-Routing-Count-Mode"} {
						assert.Empty(t, recorder.Header().Get(header))
					}
					channel, group, err := CacheGetRandomSatisfiedChannel(&RetryParam{Ctx: ctx, ModelName: "routing-model", TokenGroup: "default", RequestPath: sample.path})
					require.NoError(t, err)
					require.NotNil(t, channel)
					assert.Equal(t, 101, channel.Id)
					assert.Equal(t, "default", group)
					require.NoError(t, AcquireManagedRoutingChannel(ctx, channel.Id))
				})
			}
		})
	}
}

func TestRoutingPoolSelectionAndCapacity(t *testing.T) {
	previousDB, previousCache := model.DB, common.MemoryCacheEnabled
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, common.MemoryCacheEnabled = database, true
	t.Cleanup(func() {
		model.DB, common.MemoryCacheEnabled = previousDB, previousCache
		if previousDB != nil && previousCache {
			model.InitChannelCache()
		}
	})
	require.NoError(t, database.AutoMigrate(&model.Channel{}, &model.Ability{}))
	for _, id := range []int{101, 102, 103} {
		require.NoError(t, database.Create(&model.Channel{Id: id, Type: 1, Status: common.ChannelStatusEnabled, Models: "routing-model", Group: "default"}).Error)
		require.NoError(t, database.Create(&model.Ability{ChannelId: id, Model: "routing-model", Group: "default", Enabled: true}).Error)
	}
	model.InitChannelCache()
	first := operation_setting.RoutingInstance{ID: t.Name() + "-s1", ChannelIDs: []int{101}, MaxInflight: 2, WorkCapacity: 10000}
	second := operation_setting.RoutingInstance{ID: t.Name() + "-s2", ChannelIDs: []int{102}, MaxInflight: 2, WorkCapacity: 10000}
	long := operation_setting.RoutingInstance{ID: t.Name() + "-long", ChannelIDs: []int{103}, MaxInflight: 1, WorkCapacity: 100000}
	rule := operation_setting.RoutingRule{ID: "test", ShortPool: []operation_setting.RoutingInstance{first, second}, LongPool: []operation_setting.RoutingInstance{long}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{}`))
	state := &routingRequest{config: &operation_setting.RequestRouting{StateBackend: "local", Version: "test"}, rule: &rule, pool: rule.ShortPool, work: 100}
	ctx.Set(routingContextKey, state)
	t.Cleanup(func() { ReleaseRoutingAttempt(ctx) })
	_, acquired, err := routingState(context.Background(), "local", first, "occupied", 9000, "acquire")
	require.NoError(t, err)
	require.True(t, acquired)
	t.Cleanup(func() { _, _, _ = routingState(context.Background(), "local", first, "occupied", 0, "release") })
	param := &RetryParam{Ctx: ctx, ModelName: "routing-model", TokenGroup: "default", RequestPath: ctx.Request.URL.Path}
	channel, group, err := selectRoutedChannel(param)
	require.NoError(t, err)
	assert.Equal(t, 102, channel.Id, "less loaded short instance must win")
	assert.Equal(t, "default", group)
	reused, _, err := selectRoutedChannel(param)
	require.NoError(t, err)
	assert.Equal(t, channel.Id, reused.Id)
	load, _, err := routingState(context.Background(), "local", second, "", 0, "read")
	require.NoError(t, err)
	assert.Equal(t, 1, load.inflight, "initial selection must reuse its reservation for the first upstream attempt")
	ReleaseRoutingAttempt(ctx)
	state.pool, state.class = rule.LongPool, "long"
	channel, _, err = selectRoutedChannel(param)
	require.NoError(t, err)
	assert.Equal(t, 103, channel.Id, "long requests must stay in the long pool")
	ReleaseRoutingAttempt(ctx)
	_, acquired, err = routingState(context.Background(), "local", long, "occupied", 9000, "acquire")
	require.NoError(t, err)
	require.True(t, acquired)
	t.Cleanup(func() { _, _, _ = routingState(context.Background(), "local", long, "occupied", 0, "release") })
	_, _, err = selectRoutedChannel(param)
	require.Error(t, err)
	var routingErr *RoutingError
	require.ErrorAs(t, err, &routingErr)
	assert.Equal(t, 429, routingErr.Status, "full long pool must not fall back to available short instances")
}

func TestRoutingAdmissionAndIdempotentRelease(t *testing.T) {
	instance := operation_setting.RoutingInstance{ID: t.Name(), MaxInflight: 1}
	_, acquired, err := routingState(context.Background(), "local", instance, "first", 123, "acquire")
	require.NoError(t, err)
	require.True(t, acquired)
	load, acquired, err := routingState(context.Background(), "local", instance, "second", 42, "acquire")
	require.NoError(t, err)
	assert.False(t, acquired)
	assert.Equal(t, 1, load.inflight)
	assert.Equal(t, float64(123), load.work)
	for count := 0; count < 2; count++ {
		_, _, err = routingState(context.Background(), "local", instance, "first", 0, "release")
		require.NoError(t, err)
	}
	load, acquired, err = routingState(context.Background(), "local", instance, "second", 42, "acquire")
	require.NoError(t, err)
	assert.True(t, acquired)
	assert.Equal(t, 1, load.inflight)
	_, _, err = routingState(context.Background(), "local", instance, "second", 0, "release")
	require.NoError(t, err)
}

func TestRoutingConcurrentAdmission(t *testing.T) {
	instance := operation_setting.RoutingInstance{ID: t.Name(), MaxInflight: 3}
	start := make(chan struct{})
	results := make(chan bool, 12)
	var workers sync.WaitGroup
	for index := 0; index < 12; index++ {
		workers.Add(1)
		go func(attempt string) {
			defer workers.Done()
			<-start
			_, acquired, err := routingState(context.Background(), "local", instance, attempt, 1, "acquire")
			assert.NoError(t, err)
			results <- acquired
		}(fmt.Sprintf("attempt-%d", index))
	}
	close(start)
	workers.Wait()
	close(results)
	admitted := 0
	for acquired := range results {
		if acquired {
			admitted++
		}
	}
	assert.Equal(t, instance.MaxInflight, admitted)
	load, _, err := routingState(context.Background(), "local", instance, "", 0, "read")
	require.NoError(t, err)
	assert.Equal(t, instance.MaxInflight, load.inflight)
	for index := 0; index < 12; index++ {
		_, _, err := routingState(context.Background(), "local", instance, fmt.Sprintf("attempt-%d", index), 0, "release")
		require.NoError(t, err)
	}
}
