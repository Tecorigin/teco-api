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

	require.Equal(t, "现在保存的user内容太多，帮我精简。", got)
}

func TestBuildAuditPromptForRequest_OpenAIResponsesInputStringWrapped(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrapped := `<environment_context>internal context</environment_context><user_input>这是用户问题</user_input>`
	wrappedJSON, err := common.Marshal(wrapped)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.False(t, strings.HasPrefix(got, "user\n"))
	require.Contains(t, got, "这是用户问题")
	require.NotContains(t, got, "internal context")
}

func TestBuildAuditPromptForRequest_StripSystemReminderAndInstructionsFromUser(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrapped := `<system-reminder>敏感系统提醒</system-reminder><INSTRUCTIONS source="x">这是指令块</INSTRUCTIONS><user_input>真正的用户问题</user_input>`
	wrappedJSON, err := common.Marshal(wrapped)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "真正的用户问题", got)
}

func TestBuildAuditPromptForRequest_OpenAIResponsesInputStringUserRequestOnly(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrapped := `<reminderInstructions>*****</reminderInstructions>
<userRequest>
你好啊，你是谁
</userRequest>
<context>****</context>
<userRequest>
再介绍一下你能做什么
</userRequest>`
	wrappedJSON, err := common.Marshal(wrapped)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "你好啊，你是谁\n再介绍一下你能做什么", got)
}

func TestBuildAuditPromptFromRawText_BeforeTruncation(t *testing.T) {
	raw := `<INSTRUCTIONS>系统说明</INSTRUCTIONS>用户问题在前面。`

	got := BuildAuditPromptFromRawText(raw)

	require.Equal(t, "用户问题在前面。", got)
}
