package operation_setting

import (
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const RequestRoutingOption = "RequestRouting"

type RoutingInstance struct {
	ID           string  `json:"instance_id"`
	ChannelIDs   []int   `json:"channel_ids"`
	MaxInflight  int     `json:"max_inflight,omitempty"`
	WorkCapacity float64 `json:"-"`
}

type RoutingRule struct {
	ID        string            `json:"id"`
	Models    []string          `json:"models"`
	Groups    []string          `json:"groups"`
	Paths     []string          `json:"paths"`
	Threshold int               `json:"input_token_threshold"`
	ShortPool []RoutingInstance `json:"short_pool"`
	LongPool  []RoutingInstance `json:"long_pool"`
	Ranges    []RoutingRange    `json:"ranges"`
}

type RoutingRange struct {
	MinTokens int               `json:"min_tokens"`
	MaxTokens *int              `json:"max_tokens,omitempty"`
	Instances []RoutingInstance `json:"instances"`
}

func (rule RoutingRule) InstancePools() [][]RoutingInstance {
	pools := make([][]RoutingInstance, 0, len(rule.Ranges))
	for _, interval := range rule.Ranges {
		pools = append(pools, interval.Instances)
	}
	return pools
}

type RequestRouting struct {
	Enabled      bool          `json:"enabled"`
	Version      string        `json:"version"`
	StateBackend string        `json:"state_backend"`
	Rules        []RoutingRule `json:"rules"`
}

func ParseRequestRouting(raw string) (*RequestRouting, error) {
	config := &RequestRouting{StateBackend: "redis"}
	if strings.TrimSpace(raw) == "" {
		return config, nil
	}
	if len(raw) > 262144 {
		return nil, fmt.Errorf("routing configuration exceeds 256 KiB")
	}
	if err := common.UnmarshalJsonStr(raw, config); err != nil {
		return nil, err
	}
	config.StateBackend = "redis"
	if config.Enabled && len(config.Rules) == 0 {
		return nil, fmt.Errorf("enabled routing requires rules")
	}
	if len(config.Rules) > 100 {
		return nil, fmt.Errorf("at most 100 routing rules are allowed")
	}
	identities := map[string]RoutingInstance{}
	instancePools := map[string]int{}
	channels := map[int]string{}
	ruleIDs := map[string]bool{}
	matches := map[string]bool{}
	for ruleIndex := range config.Rules {
		rule := &config.Rules[ruleIndex]
		rule.Paths = []string{"/v1/chat/completions", "/v1/messages", "/v1/responses", "/pg/chat/completions"}
		if len(rule.Ranges) == 0 && rule.Threshold > 0 {
			threshold := rule.Threshold
			rule.Ranges = []RoutingRange{{MinTokens: 0, MaxTokens: &threshold, Instances: rule.ShortPool}, {MinTokens: threshold, Instances: rule.LongPool}}
		}
		rule.Threshold, rule.ShortPool, rule.LongPool = 0, nil, nil
		if rule.ID == "" || ruleIDs[rule.ID] {
			return nil, fmt.Errorf("rule IDs must be nonempty and unique")
		}
		ruleIDs[rule.ID] = true
		if len(rule.Models) == 0 || len(rule.Ranges) == 0 || len(rule.Ranges) > 100 {
			return nil, fmt.Errorf("rule %s requires models and token ranges", rule.ID)
		}
		groups := rule.Groups
		if len(groups) == 0 {
			groups = []string{"*"}
		}
		for _, path := range rule.Paths {
			if path != "/v1/chat/completions" && path != "/v1/messages" && path != "/v1/responses" && path != "/pg/chat/completions" {
				return nil, fmt.Errorf("unsupported routing path: %s", path)
			}
			for _, group := range groups {
				for _, name := range rule.Models {
					if group == "" || name == "" {
						return nil, fmt.Errorf("empty group or model")
					}
					key := group + "\x00" + name + "\x00" + path
					conflict := matches[key] || matches["*"+"\x00"+name+"\x00"+path]
					if group == "*" {
						for existing := range matches {
							if strings.HasSuffix(existing, "\x00"+name+"\x00"+path) {
								conflict = true
								break
							}
						}
					}
					if conflict {
						return nil, fmt.Errorf("overlapping routing rules for %s", name)
					}
					matches[key] = true
				}
			}
		}
		poolIDs := map[string]bool{}
		for poolIndex, interval := range rule.Ranges {
			if interval.MinTokens < 0 || interval.MinTokens > 10000000 || (interval.MaxTokens != nil && (*interval.MaxTokens <= interval.MinTokens || *interval.MaxTokens > 10000000)) {
				return nil, fmt.Errorf("rule %s has invalid token range", rule.ID)
			}
			for previousIndex := 0; previousIndex < poolIndex; previousIndex++ {
				previous := rule.Ranges[previousIndex]
				if (previous.MaxTokens == nil || interval.MinTokens < *previous.MaxTokens) && (interval.MaxTokens == nil || previous.MinTokens < *interval.MaxTokens) {
					return nil, fmt.Errorf("rule %s has overlapping token ranges", rule.ID)
				}
			}
			pool := interval.Instances
			for instanceIndex := range pool {
				pool[instanceIndex].WorkCapacity = 100000
			}
			if len(pool) == 0 || len(pool) > 100 {
				return nil, fmt.Errorf("rule %s requires two nonempty bounded pools", rule.ID)
			}
			for _, instance := range pool {
				if instance.ID == "" || len(instance.ID) > 100 || strings.ContainsAny(instance.ID, "{}\x00") || poolIDs[instance.ID] {
					return nil, fmt.Errorf("duplicate or invalid instance in rule %s", rule.ID)
				}
				poolIDs[instance.ID] = true
				if previousPool, exists := instancePools[instance.ID]; exists && previousPool != ruleIndex*100+poolIndex {
					return nil, fmt.Errorf("instance %s cannot belong to multiple routing pools", instance.ID)
				}
				instancePools[instance.ID] = ruleIndex*100 + poolIndex
				if instance.MaxInflight < 0 || instance.MaxInflight > 10000 || instance.WorkCapacity <= 0 || instance.WorkCapacity > 1e12 || math.IsNaN(instance.WorkCapacity) || math.IsInf(instance.WorkCapacity, 0) || len(instance.ChannelIDs) == 0 || len(instance.ChannelIDs) > 100 {
					return nil, fmt.Errorf("invalid capacity for %s", instance.ID)
				}
				if previous, ok := identities[instance.ID]; ok && (previous.MaxInflight != instance.MaxInflight || previous.WorkCapacity != instance.WorkCapacity) {
					return nil, fmt.Errorf("inconsistent capacity for %s", instance.ID)
				}
				identities[instance.ID] = instance
				seen := map[int]bool{}
				for _, channelID := range instance.ChannelIDs {
					if channelID < 1 || seen[channelID] {
						return nil, fmt.Errorf("invalid or duplicate channel ID")
					}
					seen[channelID] = true
					if owner, ok := channels[channelID]; ok && owner != instance.ID {
						return nil, fmt.Errorf("channel %d belongs to multiple instances", channelID)
					}
					channels[channelID] = instance.ID
				}
			}
		}
	}
	return config, nil
}
