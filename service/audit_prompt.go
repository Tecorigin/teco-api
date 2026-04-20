package service

import (
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
		case "user", "":
			appendIfNotEmpty(userTexts, text)
		case "system", "developer":
			appendIfNotEmpty(systemTexts, text)
		default:
			appendIfNotEmpty(otherTexts, text)
		}
	}
}

func openAIMessageText(message dto.Message) string {
	if message.IsStringContent() {
		return strings.TrimSpace(message.StringContent())
	}
	parts := make([]string, 0)
	for _, content := range message.ParseContent() {
		if content.Type == dto.ContentTypeText {
			appendIfNotEmpty(&parts, content.Text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func claudeMessageText(message dto.ClaudeMessage) string {
	if message.IsStringContent() {
		return strings.TrimSpace(message.GetStringContent())
	}
	parts := make([]string, 0)
	contents, _ := message.ParseContent()
	for _, content := range contents {
		if content.Type == "text" {
			appendIfNotEmpty(&parts, content.GetText())
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func anyToStrings(v any) []string {
	switch item := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(item) == "" {
			return nil
		}
		return []string{item}
	case []any:
		out := make([]string, 0, len(item))
		for _, val := range item {
			s := strings.TrimSpace(fmt.Sprintf("%v", val))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		s := strings.TrimSpace(fmt.Sprintf("%v", item))
		if s == "" {
			return nil
		}
		return []string{s}
	}
}

func appendIfNotEmpty(values *[]string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	*values = append(*values, trimmed)
}
