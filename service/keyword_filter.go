package service

import (
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting"
)

type KeywordFilterResult struct {
	Changed bool
	Hits    []string
}

func ApplyKeywordFilterOnRequest(request dto.Request) KeywordFilterResult {
	if request == nil || !setting.ShouldApplyKeywordFilter() {
		return KeywordFilterResult{}
	}

	result := KeywordFilterResult{}
	applyText := func(text string) string {
		changed, hits, out := replaceByRules(text, setting.KeywordReplaceRules)
		if changed {
			result.Changed = true
			result.Hits = append(result.Hits, hits...)
		}
		return out
	}

	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		replaceGeneralOpenAIRequest(req, applyText)
	case *dto.ClaudeRequest:
		replaceClaudeRequest(req, applyText)
	case *dto.GeminiChatRequest:
		replaceGeminiChatRequest(req, applyText)
	case *dto.OpenAIResponsesRequest:
		replaceOpenAIResponsesRequest(req, applyText)
	case *dto.AudioRequest:
		req.Input = applyText(req.Input)
	case *dto.ImageRequest:
		req.Prompt = applyText(req.Prompt)
	case *dto.EmbeddingRequest:
		replaceEmbeddingRequest(req, applyText)
	}

	result.Hits = dedupStrings(result.Hits)
	return result
}

func replaceGeneralOpenAIRequest(req *dto.GeneralOpenAIRequest, apply func(string) string) {
	if req == nil {
		return
	}
	req.Instruction = apply(req.Instruction)
	req.Messages = replaceMessages(req.Messages, apply)
	req.Prompt = replaceStringAny(req.Prompt, apply)
	req.Input = replaceStringAny(req.Input, apply)
}

func replaceClaudeRequest(req *dto.ClaudeRequest, apply func(string) string) {
	if req == nil {
		return
	}
	req.Prompt = apply(req.Prompt)
	if req.IsStringSystem() {
		req.SetStringSystem(apply(req.GetStringSystem()))
	} else {
		systemMedia := req.ParseSystem()
		for i := range systemMedia {
			systemMedia[i].SetText(apply(systemMedia[i].GetText()))
		}
		req.System = systemMedia
	}
	for i := range req.Messages {
		if req.Messages[i].IsStringContent() {
			req.Messages[i].SetStringContent(apply(req.Messages[i].GetStringContent()))
			continue
		}
		contents, _ := req.Messages[i].ParseContent()
		for j := range contents {
			if contents[j].Type == "text" {
				contents[j].SetText(apply(contents[j].GetText()))
			}
		}
		req.Messages[i].SetContent(contents)
	}
}

func replaceGeminiChatRequest(req *dto.GeminiChatRequest, apply func(string) string) {
	if req == nil {
		return
	}
	for i := range req.Contents {
		for j := range req.Contents[i].Parts {
			req.Contents[i].Parts[j].Text = apply(req.Contents[i].Parts[j].Text)
		}
	}
	if req.SystemInstructions != nil {
		for i := range req.SystemInstructions.Parts {
			req.SystemInstructions.Parts[i].Text = apply(req.SystemInstructions.Parts[i].Text)
		}
	}
}

func replaceOpenAIResponsesRequest(req *dto.OpenAIResponsesRequest, apply func(string) string) {
	if req == nil {
		return
	}
	req.Input = replaceStringInRawJSON(req.Input, apply)
	req.Instructions = replaceStringInRawJSON(req.Instructions, apply)
	req.Prompt = replaceStringInRawJSON(req.Prompt, apply)
}

func replaceEmbeddingRequest(req *dto.EmbeddingRequest, apply func(string) string) {
	if req == nil {
		return
	}
	req.Input = replaceStringAny(req.Input, apply)
}

func replaceMessages(messages []dto.Message, apply func(string) string) []dto.Message {
	for i := range messages {
		if messages[i].IsStringContent() {
			messages[i].SetStringContent(apply(messages[i].StringContent()))
			continue
		}
		arrayContent := messages[i].ParseContent()
		for j := range arrayContent {
			if arrayContent[j].Type == dto.ContentTypeText {
				arrayContent[j].Text = apply(arrayContent[j].Text)
			}
		}
		messages[i].Content = arrayContent
	}
	return messages
}

func replaceStringAny(v any, apply func(string) string) any {
	switch item := v.(type) {
	case nil:
		return v
	case string:
		return apply(item)
	case []any:
		out := make([]any, len(item))
		for i := range item {
			out[i] = replaceStringAny(item[i], apply)
		}
		return out
	case map[string]any:
		for key, val := range item {
			item[key] = replaceStringAny(val, apply)
		}
		return item
	default:
		return v
	}
}

func replaceStringInRawJSON(raw []byte, apply func(string) string) []byte {
	if len(raw) == 0 {
		return raw
	}
	var parsed any
	if err := common.Unmarshal(raw, &parsed); err != nil {
		return raw
	}
	parsed = replaceStringAny(parsed, apply)
	b, err := common.Marshal(parsed)
	if err != nil {
		return raw
	}
	return b
}

func replaceByRules(text string, rules []setting.KeywordReplaceRule) (bool, []string, string) {
	if text == "" || len(rules) == 0 {
		return false, nil, text
	}
	output := text
	hits := make([]string, 0)
	changed := false
	for _, rule := range rules {
		from := strings.TrimSpace(rule.From)
		if from == "" {
			continue
		}

		pattern, ok := buildReplacePattern(from)
		if !ok {
			continue
		}

		if !pattern.MatchString(output) {
			continue
		}
		hits = append(hits, from)
		output = pattern.ReplaceAllString(output, rule.To)
		changed = true
	}
	return changed, hits, output
}

func buildReplacePattern(from string) (*regexp.Regexp, bool) {
	trimmed := strings.TrimSpace(from)
	if trimmed == "" {
		return nil, false
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "re:") {
		rawPattern := strings.TrimSpace(trimmed[3:])
		if rawPattern == "" {
			return nil, false
		}
		re, err := regexp.Compile("(?i)" + rawPattern)
		if err != nil {
			return nil, false
		}
		return re, true
	}

	if strings.ContainsAny(trimmed, "*?") {
		wildcard := regexp.QuoteMeta(trimmed)
		wildcard = strings.ReplaceAll(wildcard, `\*`, `.*`)
		wildcard = strings.ReplaceAll(wildcard, `\?`, `.`)
		re, err := regexp.Compile("(?i)" + wildcard)
		if err != nil {
			return nil, false
		}
		return re, true
	}

	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(trimmed))
	if err != nil {
		return nil, false
	}
	return re, true
}

func dedupStrings(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
