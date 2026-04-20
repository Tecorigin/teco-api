package setting

import "strings"

var CheckSensitiveEnabled = true
var CheckSensitiveOnPromptEnabled = true
var KeywordFilterEnabled = false

//var CheckSensitiveOnCompletionEnabled = true

// StopOnSensitiveEnabled 如果检测到敏感词，是否立刻停止生成，否则替换敏感词
var StopOnSensitiveEnabled = true

// StreamCacheQueueLength 流模式缓存队列长度，0表示无缓存
var StreamCacheQueueLength = 0

// SensitiveWords 敏感词
// var SensitiveWords []string
var SensitiveWords = []string{
	"test_sensitive",
}

type KeywordReplaceRule struct {
	From string
	To   string
}

var KeywordReplaceRules []KeywordReplaceRule

func SensitiveWordsToString() string {
	return strings.Join(SensitiveWords, "\n")
}

func SensitiveWordsFromString(s string) {
	SensitiveWords = []string{}
	sw := strings.Split(s, "\n")
	for _, w := range sw {
		w = strings.TrimSpace(w)
		if w != "" {
			SensitiveWords = append(SensitiveWords, w)
		}
	}
}

func KeywordReplaceRulesToString() string {
	if len(KeywordReplaceRules) == 0 {
		return ""
	}
	lines := make([]string, 0, len(KeywordReplaceRules))
	for _, rule := range KeywordReplaceRules {
		from := strings.TrimSpace(rule.From)
		if from == "" {
			continue
		}
		lines = append(lines, from+"=>"+rule.To)
	}
	return strings.Join(lines, "\n")
}

func KeywordReplaceRulesFromString(s string) {
	KeywordReplaceRules = []KeywordReplaceRule{}
	if s == "" {
		return
	}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=>", 2)
		if len(parts) != 2 {
			continue
		}
		from := strings.TrimSpace(parts[0])
		to := strings.TrimSpace(parts[1])
		if from == "" {
			continue
		}
		KeywordReplaceRules = append(KeywordReplaceRules, KeywordReplaceRule{From: from, To: to})
	}
}

func ShouldCheckPromptSensitive() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnPromptEnabled
}

func ShouldApplyKeywordFilter() bool {
	return KeywordFilterEnabled && len(KeywordReplaceRules) > 0
}

//func ShouldCheckCompletionSensitive() bool {
//	return CheckSensitiveEnabled && CheckSensitiveOnCompletionEnabled
//}
