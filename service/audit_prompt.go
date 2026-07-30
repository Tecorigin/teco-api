package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
    "unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

const auditPromptFallbackMaxRunes = 500

type userAuditAccumulator struct {
	tagged   []string
	fallback string
}

// BuildAuditPromptForRequest builds an audit-friendly prompt text and prioritizes user input.
func BuildAuditPromptForRequest(request dto.Request) string {
	if request == nil {
		return ""
	}

	userAccumulator := &userAuditAccumulator{}
	var systemTexts []string
	var otherTexts []string

	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		collectOpenAITexts(req, userAccumulator, &systemTexts, &otherTexts)
	case *dto.OpenAIResponsesRequest:
		collectOpenAIResponsesTexts(req, userAccumulator, &systemTexts, &otherTexts)
	case *dto.ClaudeRequest:
		collectClaudeTexts(req, userAccumulator, &systemTexts, &otherTexts)
	case *dto.GeminiChatRequest:
		collectGeminiTexts(req, userAccumulator, &systemTexts, &otherTexts)
	}

	userTexts := userAccumulator.Result()
	if len(userTexts) > 0 {
		return strings.TrimSpace(strings.Join(userTexts, "\n"))
	}
	if len(systemTexts) > 0 {
		return ""
	}
	if len(otherTexts) > 0 {
		return strings.TrimSpace(strings.Join(otherTexts, "\n"))
	}
	return ""
}

func collectOpenAITexts(req *dto.GeneralOpenAIRequest, userAccumulator *userAuditAccumulator, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	if req.Prompt != nil {
		for _, text := range anyToStrings(req.Prompt) {
			userAccumulator.Add(text)
		}
	}

	if req.Input != nil {
		for _, text := range req.ParseInput() {
			userAccumulator.Add(text)
		}
	}

	for _, message := range req.Messages {
		text := openAIMessageText(message)
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "user":
			userAccumulator.Add(text)
		case "system", "developer":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func collectOpenAIResponsesTexts(req *dto.OpenAIResponsesRequest, userAccumulator *userAuditAccumulator, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	if len(req.Instructions) > 0 {
		appendIfNotEmpty(systemTexts, jsonRawToString(req.Instructions))
	}

	if len(req.Prompt) > 0 {
		userAccumulator.Add(jsonRawToString(req.Prompt))
	}

	if req.Input == nil {
		return
	}

	if common.GetJsonType(req.Input) == "string" {
		var text string
		_ = common.Unmarshal(req.Input, &text)
		userAccumulator.Add(text)
		for _, t := range extractSystemTextsFromInputText(text) {
			appendIfNotEmpty(systemTexts, t)
		}
		return
	}

	if common.GetJsonType(req.Input) != "array" {
		for _, input := range req.ParseInput() {
			switch input.Type {
			case "input_text":
				userAccumulator.Add(input.Text)
				for _, text := range extractSystemTextsFromInputText(input.Text) {
					appendIfNotEmpty(systemTexts, text)
				}
			case "input_file":
				appendIfNotEmpty(otherTexts, input.FileUrl)
			case "input_image":
				appendIfNotEmpty(otherTexts, input.ImageUrl)
			}
		}
		return
	}

	var inputs []dto.Input
	_ = common.Unmarshal(req.Input, &inputs)
	for _, input := range inputs {
		role := strings.ToLower(strings.TrimSpace(input.Role))
		for _, media := range parseResponsesInputContent(input.Content) {
			switch media.Type {
			case "input_text":
				if role == "system" || role == "developer" || role == "model" || role == "assistant" {
					for _, t := range extractSystemTextsFromInputText(media.Text) {
						appendIfNotEmpty(systemTexts, t)
					}
					if cleaned := stripNonUserWrappedSections(media.Text); cleaned != "" {
						appendIfNotEmpty(systemTexts, cleaned)
					}
					continue
				}
				userAccumulator.Add(media.Text)
				for _, t := range extractSystemTextsFromInputText(media.Text) {
					appendIfNotEmpty(systemTexts, t)
				}
			case "input_file":
				appendIfNotEmpty(otherTexts, media.FileUrl)
			case "input_image":
				appendIfNotEmpty(otherTexts, media.ImageUrl)
			}
		}
	}
}

func parseResponsesInputContent(content json.RawMessage) []dto.MediaInput {
	if len(content) == 0 {
		return nil
	}

	result := make([]dto.MediaInput, 0)
	contentType := common.GetJsonType(content)
	if contentType == "string" {
		var text string
		_ = common.Unmarshal(content, &text)
		appendMediaIfNotEmpty(&result, dto.MediaInput{Type: "input_text", Text: text})
		return result
	}
	if contentType != "array" {
		return nil
	}

	var array []any
	_ = common.Unmarshal(content, &array)
	for _, itemAny := range array {
		item, ok := itemAny.(map[string]any)
		if !ok {
			continue
		}
		typeVal, ok := item["type"].(string)
		if !ok {
			continue
		}
		switch typeVal {
		case "input_text":
			text, _ := item["text"].(string)
			appendMediaIfNotEmpty(&result, dto.MediaInput{Type: "input_text", Text: text})
		case "input_image":
			var imageUrl string
			switch v := item["image_url"].(type) {
			case string:
				imageUrl = v
			case map[string]any:
				if url, ok := v["url"].(string); ok {
					imageUrl = url
				}
			}
			result = append(result, dto.MediaInput{Type: "input_image", ImageUrl: imageUrl})
		case "input_file":
			var fileUrl string
			switch v := item["file_url"].(type) {
			case string:
				fileUrl = v
			case map[string]any:
				if url, ok := v["url"].(string); ok {
					fileUrl = url
				}
			}
			result = append(result, dto.MediaInput{Type: "input_file", FileUrl: fileUrl})
		}
	}
	return result
}

func collectClaudeTexts(req *dto.ClaudeRequest, userAccumulator *userAuditAccumulator, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	userAccumulator.Add(req.Prompt)

	if req.System != nil {
		if req.IsStringSystem() {
			appendIfNotEmpty(systemTexts, req.GetStringSystem())
		} else {
			systemMedia := req.ParseSystem()
			for _, media := range systemMedia {
				if media.Type == "text" {
					appendIfNotEmpty(systemTexts, media.GetText())
				}
			}
		}
	}

	for _, message := range req.Messages {
		text := claudeMessageText(message)
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "user":
			userAccumulator.Add(text)
		case "system", "developer":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func collectGeminiTexts(req *dto.GeminiChatRequest, userAccumulator *userAuditAccumulator, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	if req.SystemInstructions != nil {
		for _, part := range req.SystemInstructions.Parts {
			appendIfNotEmpty(systemTexts, part.Text)
		}
	}

	for _, content := range req.Contents {
		parts := make([]string, 0, len(content.Parts))
		for _, part := range content.Parts {
			appendIfNotEmpty(&parts, part.Text)
		}
		text := strings.Join(parts, "\n")
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(content.Role)) {
		case "user":
			userAccumulator.Add(text)
		case "system", "developer", "model":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func openAIMessageText(message dto.Message) string {
	parts := make([]string, 0)
	if message.IsStringContent() {
		appendIfNotEmpty(&parts, message.StringContent())
	} else {
		for _, media := range message.ParseContent() {
			switch media.Type {
			case dto.ContentTypeText:
				appendIfNotEmpty(&parts, media.Text)
			case dto.ContentTypeFile:
				if file := media.GetFile(); file != nil {
					appendIfNotEmpty(&parts, file.FileName)
					appendIfNotEmpty(&parts, file.FileId)
				}
			}
		}
	}
	appendIfNotEmpty(&parts, derefString(message.ReasoningContent))
	appendIfNotEmpty(&parts, derefString(message.Reasoning))
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func claudeMessageText(message dto.ClaudeMessage) string {
	parts := make([]string, 0)
	if message.IsStringContent() {
		appendIfNotEmpty(&parts, message.GetStringContent())
	} else {
		mediaList, err := message.ParseContent()
		if err == nil {
			for _, media := range mediaList {
				switch media.Type {
				case "text":
					appendIfNotEmpty(&parts, media.GetText())
				case "tool_use":
					appendIfNotEmpty(&parts, media.Name)
					if media.Input != nil {
						appendIfNotEmpty(&parts, fmt.Sprintf("%v", media.Input))
					}
				case "tool_result":
					if media.Content != nil {
						appendIfNotEmpty(&parts, fmt.Sprintf("%v", media.Content))
					}
				}
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

var (
	userInputTagPattern          = regexp.MustCompile(`(?is)<user_input>(.*?)</user_input>`)
	userRequestTagPattern        = regexp.MustCompile(`(?is)<userRequest>(.*?)</userRequest>`)
	assistantResponseTagPattern  = regexp.MustCompile(`(?is)<assistant_response>(.*?)</assistant_response>`)
	environmentContextTagPattern = regexp.MustCompile(`(?is)<environment_context>(.*?)</environment_context>`)
	contextTagPattern            = regexp.MustCompile(`(?is)<context>(.*?)</context>`)
	reminderInstructionsPattern  = regexp.MustCompile(`(?is)<reminderInstructions>(.*?)</reminderInstructions>`)
	systemReminderTagPattern     = regexp.MustCompile(`(?is)<system-reminder>(.*?)</system-reminder>`)
	toolInstructionsTagPattern   = regexp.MustCompile(`(?is)<tool_instructions>(.*?)</tool_instructions>`)
	instructionsTagPattern       = regexp.MustCompile(`(?is)<instructions(?:\s[^>]*)?>(.*?)</instructions>`)
	standaloneRegexFragmentLine  = regexp.MustCompile(`(?m)^\s*\(\.\*\?\)\s*$`)
)

// BuildAuditPromptFromRawText preprocesses full raw context text into audit-friendly sections.
// This is used before any truncation, especially for fallback paths using CombineText.
func BuildAuditPromptFromRawText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	userTexts := extractUserTextsFromInputText(trimmed)
	if len(userTexts) > 0 {
		return strings.TrimSpace(strings.Join(userTexts, "\n"))
	}
	return ""
}

func extractUserTextsFromInputText(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if texts := extractNestedTagTexts(trimmed, "userRequest"); len(texts) > 0 {
		return texts
	}
	if texts := extractNestedTagTexts(trimmed, "user_input"); len(texts) > 0 {
		return texts
	}
	return nil
}

func (a *userAuditAccumulator) Add(text string) {
	if a == nil {
		return
	}
	tagged, fallback := splitUserAuditText(text)
	if len(tagged) > 0 {
		a.tagged = append(a.tagged, tagged...)
	}
	if fallback != "" {
		a.fallback = fallback
	}
}

func (a *userAuditAccumulator) Result() []string {
	if a == nil {
		return nil
	}
	if len(a.tagged) > 0 {
		return a.tagged
	}
	if a.fallback == "" {
		return nil
	}
	return []string{a.fallback}
}

func splitUserAuditText(text string) ([]string, string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, ""
	}
	if texts := extractUserTextsFromInputText(trimmed); len(texts) > 0 {
		return sanitizeUserAuditTexts(texts), ""
	}
	return nil, sanitizeUserAuditText(truncateAuditPromptText(stripNonUserWrappedSections(trimmed)))
}

func normalizeUserAuditText(text string) string {
	tagged, fallback := splitUserAuditText(text)
	if len(tagged) > 0 {
		return sanitizeUserAuditText(strings.TrimSpace(strings.Join(tagged, "\n")))
	}
	return fallback
}

func sanitizeUserAuditTexts(texts []string) []string {
	if len(texts) == 0 {
		return nil
	}
	result := make([]string, 0, len(texts))
	for _, text := range texts {
		cleaned := sanitizeUserAuditText(text)
		if cleaned == "" {
			continue
		}
		result = append(result, cleaned)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func sanitizeUserAuditText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	cleaned := standaloneRegexFragmentLine.ReplaceAllString(trimmed, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return ""
	}
	lines := strings.Split(cleaned, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

func truncateAuditPromptText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) <= auditPromptFallbackMaxRunes {
		return trimmed
	}
	runes := []rune(trimmed)
	return strings.TrimSpace(string(runes[:auditPromptFallbackMaxRunes]))
}

func extractSystemTextsFromInputText(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	patterns := []*regexp.Regexp{
		instructionsTagPattern,
		reminderInstructionsPattern,
		systemReminderTagPattern,
		toolInstructionsTagPattern,
		environmentContextTagPattern,
		contextTagPattern,
	}
	result := make([]string, 0)
	for _, pattern := range patterns {
		result = append(result, extractTaggedTexts(trimmed, pattern)...)
	}
	return result
}

func extractTaggedTexts(text string, pattern *regexp.Regexp) []string {
	matches := pattern.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		appendIfNotEmpty(&result, match[1])
	}
	return result
}

func extractNestedTagTexts(text string, tagName string) []string {
	type tagSpan struct {
		start      int
		contentFrom int
	}

	openTagPattern := regexp.MustCompile(`(?is)<` + regexp.QuoteMeta(tagName) + `>`)
	closeTagPattern := regexp.MustCompile(`(?is)</` + regexp.QuoteMeta(tagName) + `>`)
	boundaries := make([]struct {
		index  int
		isOpen bool
		end    int
	}, 0)
	for _, loc := range openTagPattern.FindAllStringIndex(text, -1) {
		boundaries = append(boundaries, struct {
			index  int
			isOpen bool
			end    int
		}{index: loc[0], isOpen: true, end: loc[1]})
	}
	for _, loc := range closeTagPattern.FindAllStringIndex(text, -1) {
		boundaries = append(boundaries, struct {
			index  int
			isOpen bool
			end    int
		}{index: loc[0], isOpen: false, end: loc[1]})
	}
	if len(boundaries) == 0 {
		return nil
	}
	sort.Slice(boundaries, func(i, j int) bool {
		if boundaries[i].index == boundaries[j].index {
			return boundaries[i].isOpen && !boundaries[j].isOpen
		}
		return boundaries[i].index < boundaries[j].index
	})

	stack := make([]tagSpan, 0)
	result := make([]string, 0)
	for _, boundary := range boundaries {
		if boundary.isOpen {
			stack = append(stack, tagSpan{start: boundary.index, contentFrom: boundary.end})
			continue
		}
		if len(stack) == 0 {
			continue
		}
		span := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		content := strings.TrimSpace(text[span.contentFrom:boundary.index])
		if content == "" {
			continue
		}
		if nested := extractNestedTagTexts(content, tagName); len(nested) > 0 {
			result = append(result, nested...)
			continue
		}
		cleaned := stripNonUserWrappedSections(content)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	if len(result) == 0 {
		return nil
	}
	unique := make([]string, 0, len(result))
	seen := make(map[string]struct{}, len(result))
	for _, item := range result {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func stripNonUserWrappedSections(text string) string {
	cleaned := text
	closedPatterns := []*regexp.Regexp{
		assistantResponseTagPattern,
		contextTagPattern,
		reminderInstructionsPattern,
		environmentContextTagPattern,
		systemReminderTagPattern,
		toolInstructionsTagPattern,
		instructionsTagPattern,
	}
	for _, pattern := range closedPatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}
	cleaned = regexp.MustCompile(`(?is)</?[^>]+>`).ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

func anyToStrings(value any) []string {
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			result = append(result, anyToStrings(item)...)
		}
		return result
	case []string:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if strings.TrimSpace(item) != "" {
				result = append(result, item)
			}
		}
		return result
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			return nil
		}
		return []string{text}
	}
}

func appendIfNotEmpty(target *[]string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	*target = append(*target, value)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func jsonRawToString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var str string
	if err := common.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str)
	}
	return strings.TrimSpace(string(raw))
}

func appendMediaIfNotEmpty(target *[]dto.MediaInput, value dto.MediaInput) {
	if strings.TrimSpace(value.Text) == "" && strings.TrimSpace(value.FileUrl) == "" && strings.TrimSpace(value.ImageUrl) == "" {
		return
	}
	*target = append(*target, value)
}
