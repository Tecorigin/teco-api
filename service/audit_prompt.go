package service

import (
	"encoding/json"
	"fmt"
	"strings"

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

	parts := make([]string, 0, 3)
	if len(userTexts) > 0 {
		parts = append(parts, "user\n"+strings.Join(userTexts, "\n"))
	}
	if len(systemTexts) > 0 {
		parts = append(parts, "system\n"+strings.Join(systemTexts, "\n"))
	}
	if len(parts) == 0 && len(otherTexts) > 0 {
		parts = append(parts, strings.Join(otherTexts, "\n"))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
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

	for _, input := range req.ParseInput() {
		switch input.Type {
		case "input_text":
			appendIfNotEmpty(userTexts, input.Text)
		case "input_file":
			appendIfNotEmpty(otherTexts, input.FileUrl)
		case "input_image":
			appendIfNotEmpty(otherTexts, input.ImageUrl)
		}
	}
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
	if err := json.Unmarshal(raw, &str); err == nil {
		return strings.TrimSpace(str)
	}
	return strings.TrimSpace(string(raw))
}
