package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const routingContextKey = "request_routing_state"

type RoutingError struct {
	Status int
	Code   string
}

func (err *RoutingError) Error() string { return err.Code }

type routingRequest struct {
	config        *operation_setting.RequestRouting
	rule          *operation_setting.RoutingRule
	pool          []operation_setting.RoutingInstance
	class         string
	tokens        int
	work          float64
	release       func()
	selected      *model.Channel
	selectedGroup string
}

type routingLoad struct {
	inflight int
	work     float64
}

var routingLocal = struct {
	sync.Mutex
	instances map[string]map[string]float64
}{instances: map[string]map[string]float64{}}

func PrepareRequestRouting(c *gin.Context, name, group string) error {
	if _, exists := c.Get(routingContextKey); exists {
		return nil
	}
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[operation_setting.RequestRoutingOption]
	common.OptionMapRWMutex.RUnlock()
	config, err := operation_setting.ParseRequestRouting(raw)
	if err != nil {
		return &RoutingError{http.StatusServiceUnavailable, "routing_configuration_invalid"}
	}
	if !config.Enabled {
		return nil
	}
	state := &routingRequest{config: config}
	for index := range config.Rules {
		rule := &config.Rules[index]
		if slices.Contains(rule.Models, name) && (len(rule.Groups) == 0 || slices.Contains(rule.Groups, group)) && slices.Contains(rule.Paths, c.Request.URL.Path) {
			state.rule = rule
			break
		}
	}
	if state.rule != nil {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return err
		}
		body, err := storage.Bytes()
		if err != nil {
			return err
		}
		state.tokens, state.work, err = routingFeatures(body, name, c.Request.URL.Path)
		if err != nil {
			return err
		}
		for _, interval := range state.rule.Ranges {
			if state.tokens >= interval.MinTokens && (interval.MaxTokens == nil || state.tokens < *interval.MaxTokens) {
				state.pool = interval.Instances
				state.class = fmt.Sprintf("%d-unbounded", interval.MinTokens)
				if interval.MaxTokens != nil {
					state.class = fmt.Sprintf("%d-%d", interval.MinTokens, *interval.MaxTokens)
				}
				break
			}
		}
		if len(state.pool) == 0 {
			state.rule = nil
		} else {
			c.Header("X-Teco-Routing-Version", config.Version)
			c.Header("X-Teco-Routing-Class", state.class)
			c.Header("X-Teco-Routing-Count-Mode", "project-token-counter-v1")
		}
	}
	c.Set(routingContextKey, state)
	return nil
}

func routingFeatures(body []byte, modelName, path string) (int, float64, error) {
	if !gjson.ValidBytes(body) {
		return 0, 0, &RoutingError{http.StatusBadRequest, "routing_invalid_json"}
	}
	if gjson.GetBytes(body, "background").Bool() || gjson.GetBytes(body, "previous_response_id").Exists() || gjson.GetBytes(body, "conversation").Exists() {
		return 0, 0, &RoutingError{http.StatusBadRequest, "routing_stateful_or_background_unsupported"}
	}
	var root map[string]any
	if err := common.Unmarshal(body, &root); err != nil {
		return 0, 0, err
	}
	inputBytes := 0
	for _, key := range []string{"messages", "system", "instructions", "input", "tools", "tool_choice", "functions", "response_format", "text"} {
		value, exists := root[key]
		if !exists {
			continue
		}
		if key == "messages" || key == "input" || key == "system" {
			if routingUnsupportedContent(value) {
				return 0, 0, &RoutingError{http.StatusBadRequest, "routing_multimodal_unsupported"}
			}
		}
		encoded, err := common.Marshal(value)
		if err != nil {
			return 0, 0, err
		}
		inputBytes += len(encoded)
	}
	if inputBytes == 0 {
		return 0, 0, &RoutingError{http.StatusBadRequest, "routing_input_missing"}
	}
	if inputBytes > 10000000 {
		return 0, 0, &RoutingError{http.StatusRequestEntityTooLarge, "routing_input_too_large"}
	}
	var meta *types.TokenCountMeta
	extraText := []string{}
	switch path {
	case "/v1/chat/completions", "/pg/chat/completions":
		var request dto.GeneralOpenAIRequest
		if err := common.Unmarshal(body, &request); err != nil {
			return 0, 0, err
		}
		meta = request.GetTokenCountMeta()
		for _, message := range gjson.GetBytes(body, "messages").Array() {
			for _, key := range []string{"tool_calls", "function_call", "tool_call_id"} {
				if value := message.Get(key); value.Exists() {
					extraText = append(extraText, value.Raw)
				}
			}
		}
		for _, key := range []string{"functions", "tool_choice", "response_format"} {
			if value := gjson.GetBytes(body, key); value.Exists() {
				extraText = append(extraText, value.Raw)
			}
		}
	case "/v1/messages":
		var request dto.ClaudeRequest
		if err := common.Unmarshal(body, &request); err != nil {
			return 0, 0, err
		}
		meta = request.GetTokenCountMeta()
		if value := gjson.GetBytes(body, "tool_choice"); value.Exists() {
			extraText = append(extraText, value.Raw)
		}
	case "/v1/responses":
		if gjson.GetBytes(body, "prompt").Exists() {
			return 0, 0, &RoutingError{http.StatusBadRequest, "routing_stateful_or_background_unsupported"}
		}
		var request dto.OpenAIResponsesRequest
		if err := common.Unmarshal(body, &request); err != nil {
			return 0, 0, err
		}
		meta = request.GetTokenCountMeta()
		for _, item := range gjson.GetBytes(body, "input").Array() {
			switch item.Get("type").String() {
			case "function_call", "function_call_output", "input_text", "output_text":
				extraText = append(extraText, item.Raw)
			}
		}
		meta.MessagesCount = len(gjson.GetBytes(body, "input").Array())
		meta.ToolsCount = len(gjson.GetBytes(body, "tools").Array())
	default:
		return 0, 0, &RoutingError{http.StatusBadRequest, "routing_protocol_unsupported"}
	}
	if len(meta.Files) > 0 {
		return 0, 0, &RoutingError{http.StatusBadRequest, "routing_multimodal_unsupported"}
	}
	text := meta.CombineText
	if len(extraText) > 0 {
		text += "\n" + strings.Join(extraText, "\n")
	}
	inputTokens := CountTextToken(text, modelName) + meta.MessagesCount*3 + meta.NameCount*3 + meta.ToolsCount*8 + 3
	return inputTokens, float64(inputTokens), nil
}

func routingUnsupportedContent(value any) bool {
	switch content := value.(type) {
	case map[string]any:
		if kind, ok := content["type"].(string); ok && slices.Contains([]string{"image", "image_url", "input_image", "input_audio", "audio", "input_file", "file", "document", "video"}, kind) {
			return true
		}
		for _, nested := range content {
			if routingUnsupportedContent(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range content {
			if routingUnsupportedContent(nested) {
				return true
			}
		}
	}
	return false
}

func RequestRoutingActive(c *gin.Context) bool {
	value, exists := c.Get(routingContextKey)
	return exists && value.(*routingRequest).rule != nil
}

func AcquireManagedRoutingChannel(c *gin.Context, channelID int) error {
	value, exists := c.Get(routingContextKey)
	if !exists {
		return nil
	}
	state := value.(*routingRequest)
	if state.release != nil {
		return nil
	}
	for _, rule := range state.config.Rules {
		for _, pool := range rule.InstancePools() {
			for _, instance := range pool {
				if !slices.Contains(instance.ChannelIDs, channelID) {
					continue
				}
				storage, err := common.GetBodyStorage(c)
				if err != nil {
					return err
				}
				body, err := storage.Bytes()
				if err != nil {
					return err
				}
				_, work, err := routingFeatures(body, gjson.GetBytes(body, "model").String(), c.Request.URL.Path)
				if err != nil {
					return err
				}
				attempt := uuid.NewString()
				_, acquired, err := routingState(c.Request.Context(), state.config.StateBackend, instance, attempt, work, "acquire")
				if err != nil {
					return &RoutingError{http.StatusServiceUnavailable, "routing_state_unavailable"}
				}
				if !acquired {
					return &RoutingError{http.StatusTooManyRequests, "routing_capacity_exhausted"}
				}
				state.release = func() {
					if _, _, err := routingState(context.Background(), state.config.StateBackend, instance, attempt, 0, "release"); err != nil {
						logger.LogError(c, fmt.Sprintf("routing release failed instance=%s attempt=%s: %v", instance.ID, attempt, err))
					}
				}
				return nil
			}
		}
	}
	return nil
}

func ReleaseRoutingAttempt(c *gin.Context) {
	if value, exists := c.Get(routingContextKey); exists {
		state := value.(*routingRequest)
		if state.release != nil {
			state.release()
			state.release = nil
		}
	}
}

func routingState(ctx context.Context, backend string, instance operation_setting.RoutingInstance, attempt string, work float64, action string) (routingLoad, bool, error) {
	if backend == "local" {
		routingLocal.Lock()
		defer routingLocal.Unlock()
		entries := routingLocal.instances[instance.ID]
		if entries == nil {
			entries = map[string]float64{}
			routingLocal.instances[instance.ID] = entries
		}
		if action == "release" {
			delete(entries, attempt)
		}
		acquired := false
		if action == "acquire" && (instance.MaxInflight == 0 || len(entries) < instance.MaxInflight) {
			entries[attempt] = work
			acquired = true
		}
		load := routingLoad{inflight: len(entries)}
		for _, amount := range entries {
			load.work += amount
		}
		return load, acquired, nil
	}
	if common.RDB == nil {
		return routingLoad{}, false, errors.New("routing Redis is unavailable")
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	key := "teco:routing:{" + instance.ID + "}:attempts"
	if action == "release" {
		return routingLoad{}, false, common.RDB.HDel(bounded, key, attempt).Err()
	}
	if action == "acquire" {
		result, err := common.RDB.Eval(bounded, `if tonumber(ARGV[1]) > 0 and redis.call('HLEN',KEYS[1]) >= tonumber(ARGV[1]) then return 0 end redis.call('HSET',KEYS[1],ARGV[2],ARGV[3]) return 1`, []string{key}, instance.MaxInflight, attempt, work).Int()
		return routingLoad{}, result == 1, err
	}
	entries, err := common.RDB.HGetAll(bounded, key).Result()
	if err != nil {
		return routingLoad{}, false, err
	}
	load := routingLoad{inflight: len(entries)}
	for _, raw := range entries {
		amount, err := strconv.ParseFloat(raw, 64)
		if err != nil || amount < 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
			return routingLoad{}, false, errors.New("invalid routing load state")
		}
		load.work += amount
	}
	return load, false, nil
}

func selectRoutedChannel(param *RetryParam) (*model.Channel, string, error) {
	state := param.Ctx.MustGet(routingContextKey).(*routingRequest)
	if param.GetRetry() == 0 && state.release != nil && state.selected != nil {
		return state.selected, state.selectedGroup, nil
	}
	ReleaseRoutingAttempt(param.Ctx)
	groups := []string{param.TokenGroup}
	if param.TokenGroup == "auto" {
		groups = GetUserAutoGroup(common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup))
	}
	type candidate struct {
		channel  *model.Channel
		instance operation_setting.RoutingInstance
		score    float64
		group    string
	}
	candidates := []candidate{}
	eligible := false
	preferred, hasPreferred := GetPreferredChannelByAffinity(param.Ctx, param.ModelName, param.TokenGroup)
	for _, instance := range state.pool {
		var selected *model.Channel
		selectedGroup := ""
		for _, channelID := range instance.ChannelIDs {
			if specific := common.GetContextKeyString(param.Ctx, constant.ContextKeyTokenSpecificChannelId); specific != "" && specific != strconv.Itoa(channelID) {
				continue
			}
			channel, err := model.CacheGetChannel(channelID)
			if err != nil || channel.Status != common.ChannelStatusEnabled {
				continue
			}
			if channel.Type == constant.ChannelTypeAdvancedCustom {
				config := channel.GetOtherSettings().AdvancedCustom
				if config == nil || !config.SupportsPath(param.RequestPath) {
					continue
				}
			}
			for _, group := range groups {
				if model.IsChannelEnabledForGroupModel(group, param.ModelName, channelID) && (selected == nil || channel.GetPriority() > selected.GetPriority()) {
					selected = channel
					selectedGroup = group
				}
			}
		}
		if selected == nil {
			continue
		}
		eligible = true
		load, _, err := routingState(param.Ctx.Request.Context(), state.config.StateBackend, instance, "", 0, "read")
		if err != nil {
			return nil, param.TokenGroup, &RoutingError{http.StatusServiceUnavailable, "routing_state_unavailable"}
		}
		if instance.MaxInflight > 0 && load.inflight >= instance.MaxInflight {
			continue
		}
		score := (load.work + state.work) / instance.WorkCapacity
		if instance.MaxInflight > 0 {
			score = max(score, float64(load.inflight+1)/float64(instance.MaxInflight))
		}
		candidates = append(candidates, candidate{selected, instance, score, selectedGroup})
	}
	if !eligible {
		return nil, param.TokenGroup, &RoutingError{http.StatusServiceUnavailable, "routing_no_available_instance"}
	}
	if len(candidates) == 0 {
		return nil, param.TokenGroup, &RoutingError{http.StatusTooManyRequests, "routing_capacity_exhausted"}
	}
	rand.Shuffle(len(candidates), func(left, right int) { candidates[left], candidates[right] = candidates[right], candidates[left] })
	sort.SliceStable(candidates, func(left, right int) bool {
		if candidates[left].channel.GetPriority() != candidates[right].channel.GetPriority() {
			return candidates[left].channel.GetPriority() > candidates[right].channel.GetPriority()
		}
		return candidates[left].score < candidates[right].score
	})
	if hasPreferred {
		for index, option := range candidates {
			if option.channel.Id == preferred && option.channel.GetPriority() == candidates[0].channel.GetPriority() && option.score <= candidates[0].score*1.05 {
				candidates[0], candidates[index] = candidates[index], candidates[0]
				break
			}
		}
	}
	for _, option := range candidates {
		attempt := uuid.NewString()
		_, acquired, err := routingState(param.Ctx.Request.Context(), state.config.StateBackend, option.instance, attempt, state.work, "acquire")
		if err != nil {
			return nil, param.TokenGroup, &RoutingError{http.StatusServiceUnavailable, "routing_state_unavailable"}
		}
		if !acquired {
			continue
		}
		state.selected, state.selectedGroup = option.channel, option.group
		state.release = func() {
			if _, _, err := routingState(context.Background(), state.config.StateBackend, option.instance, attempt, 0, "release"); err != nil {
				logger.LogError(param.Ctx, fmt.Sprintf("routing release failed instance=%s attempt=%s: %v", option.instance.ID, attempt, err))
			}
		}
		if param.TokenGroup == "auto" {
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, option.group)
		}
		param.Ctx.Set("request_routing_decision", map[string]interface{}{
			"version": state.config.Version, "rule": state.rule.ID, "class": state.class,
			"input_estimate": state.tokens, "count_mode": "project-token-counter-v1",
			"instance_id": option.instance.ID, "channel_id": option.channel.Id,
			"attempt_id": attempt, "score": option.score,
		})
		logger.LogInfo(param.Ctx, fmt.Sprintf("routing version=%s rule=%s class=%s input_estimate=%d instance=%s attempt=%s score=%.4f", state.config.Version, state.rule.ID, state.class, state.tokens, option.instance.ID, attempt, option.score))
		return option.channel, option.group, nil
	}
	return nil, param.TokenGroup, &RoutingError{http.StatusTooManyRequests, "routing_capacity_exhausted"}
}
