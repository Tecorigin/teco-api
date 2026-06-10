package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/require"
)

func TestBuildAuditPromptForRequest_OpenAIResponsesInputRoleSeparation(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	inputJSON := `[
		{
			"type": "message",
			"role": "system",
			"content": [
				{
					"type": "input_text",
					"text": "Filesystem sandboxing defines which files can be read or written."
				}
			]
		},
		{
			"type": "message",
			"role": "user",
			"content": [
				{
					"type": "input_text",
					"text": "现在保存的user内容太多，帮我精简。"
				}
			]
		}
	]`
	req.Input = []byte(inputJSON)

	got := BuildAuditPromptForRequest(req)

	require.Contains(t, got, "user\n现在保存的user内容太多，帮我精简。")
	require.NotContains(t, got, "user\nFilesystem sandboxing")
	require.Contains(t, got, "system\nFilesystem sandboxing")
}

func TestBuildAuditPromptForRequest_OpenAIResponsesInputStringWrapped(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrapped := `<environment_context>internal context</environment_context><user_input>这是用户问题</user_input>`
	wrappedJSON, err := common.Marshal(wrapped)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.True(t, strings.HasPrefix(got, "user\n"))
	require.Contains(t, got, "这是用户问题")
	require.NotContains(t, got, "user\ninternal context")
	require.Contains(t, got, "system\ninternal context")
}

func TestBuildAuditPromptForRequest_StripSystemReminderAndInstructionsFromUser(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrapped := `<system-reminder>敏感系统提醒</system-reminder><INSTRUCTIONS source="x">这是指令块</INSTRUCTIONS><user_input>真正的用户问题</user_input>`
	wrappedJSON, err := common.Marshal(wrapped)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.Contains(t, got, "user\n真正的用户问题")
	require.NotContains(t, got, "user\n敏感系统提醒")
	require.NotContains(t, got, "user\n这是指令块")
	require.Contains(t, got, "system\n")
	require.Contains(t, got, "敏感系统提醒")
	require.Contains(t, got, "这是指令块")
}

func TestBuildAuditPromptFromRawText_BeforeTruncation(t *testing.T) {
	raw := `<INSTRUCTIONS>系统说明</INSTRUCTIONS>用户问题在前面。`

	got := BuildAuditPromptFromRawText(raw)

	require.Contains(t, got, "user\n用户问题在前面。")
	require.NotContains(t, got, "user\n系统说明")
	require.Contains(t, got, "system\n系统说明")
}
