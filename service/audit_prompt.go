package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// BuildAuditPromptForRequest builds an audit-friendly prompt text and prioritizes user input.
func BuildAuditPromptForRequest(request dto.Request) string {
	if request == nil {
		return ""
	}

	var userTexts []string
	var systemTexts []string
	var otherTexts []string

	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		collectOpenAITexts(req, &userTexts, &systemTexts, &otherTexts)
	case *dto.OpenAIResponsesRequest:
		collectOpenAIResponsesTexts(req, &userTexts, &systemTexts, &otherTexts)
	case *dto.ClaudeRequest:
		collectClaudeTexts(req, &userTexts, &systemTexts, &otherTexts)
	case *dto.GeminiChatRequest:
		collectGeminiTexts(req, &userTexts, &systemTexts, &otherTexts)
	}

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

func collectOpenAITexts(req *dto.GeneralOpenAIRequest, userTexts, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	if req.Prompt != nil {
		for _, text := range anyToStrings(req.Prompt) {
			appendIfNotEmpty(userTexts, text)
		}
	}

	if req.Input != nil {
		for _, text := range req.ParseInput() {
			appendIfNotEmpty(userTexts, text)
		}
	}

	for _, message := range req.Messages {
		text := openAIMessageText(message)
		if text == "" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "user":
			appendIfNotEmpty(userTexts, text)
		case "system", "developer":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func collectOpenAIResponsesTexts(req *dto.OpenAIResponsesRequest, userTexts, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	if len(req.Instructions) > 0 {
		appendIfNotEmpty(systemTexts, jsonRawToString(req.Instructions))
	}

	if len(req.Prompt) > 0 {
		appendIfNotEmpty(userTexts, jsonRawToString(req.Prompt))
	}

	if req.Input == nil {
		return
	}

	if common.GetJsonType(req.Input) == "string" {
		var text string
		_ = common.Unmarshal(req.Input, &text)
		for _, t := range extractUserTextsFromInputText(text) {
			appendIfNotEmpty(userTexts, t)
		}
		for _, t := range extractSystemTextsFromInputText(text) {
			appendIfNotEmpty(systemTexts, t)
		}
		return
	}

	if common.GetJsonType(req.Input) != "array" {
		for _, input := range req.ParseInput() {
			switch input.Type {
			case "input_text":
				for _, text := range extractUserTextsFromInputText(input.Text) {
					appendIfNotEmpty(userTexts, text)
				}
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
				for _, t := range extractUserTextsFromInputText(media.Text) {
					appendIfNotEmpty(userTexts, t)
				}
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

func collectClaudeTexts(req *dto.ClaudeRequest, userTexts, systemTexts, otherTexts *[]string) {
	if req == nil {
		return
	}

	appendIfNotEmpty(userTexts, req.Prompt)

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
			appendIfNotEmpty(userTexts, text)
		case "system", "developer":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func collectGeminiTexts(req *dto.GeminiChatRequest, userTexts, systemTexts, otherTexts *[]string) {
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
			appendIfNotEmpty(userTexts, text)
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
	if cleaned := stripNonUserWrappedSections(trimmed); cleaned != "" {
		return cleaned
	}
	return ""
}

func extractUserTextsFromInputText(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	if texts := extractTaggedTexts(trimmed, userRequestTagPattern); len(texts) > 0 {
		return texts
	}
	if texts := extractTaggedTexts(trimmed, userInputTagPattern); len(texts) > 0 {
		return texts
	}
	cleaned := stripNonUserWrappedSections(trimmed)
	if cleaned == "" {
		return nil
	}
	return []string{cleaned}
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
