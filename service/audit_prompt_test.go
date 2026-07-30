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

func TestBuildAuditPromptForRequest_OpenAIMessagesUserContentTaggedOnly(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "system", Content: `<instructions>系统内容</instructions>`},
			{Role: "user", Content: `<environment_context>上下文</environment_context><userRequest>你好</userRequest>`},
			{Role: "user", Content: `<user_input>今天星期几</user_input>`},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "你好\n今天星期几", got)
}

func TestBuildAuditPromptForRequest_OpenAIMessagesUserContentFallbackLastText(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: "前一条问题"},
			{Role: "user", Content: []any{map[string]any{"type": "text", "text": "今天星期几"}}},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "今天星期几", got)
}

func TestBuildAuditPromptForRequest_OpenAIMessagesTaggedPreferredOverLaterFallback(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: `<userRequest>第一个问题</userRequest>`},
			{Role: "user", Content: "后续很长的表格内容不该覆盖标签内容"},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "第一个问题", got)
}

func TestNormalizeUserAuditText_FallbackTruncates(t *testing.T) {
	longText := strings.Repeat("数", auditPromptFallbackMaxRunes+20)

	got := normalizeUserAuditText(longText)

	require.Len(t, []rune(got), auditPromptFallbackMaxRunes)
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

	require.Equal(t, "", got)
}

func TestBuildAuditPromptFromRawText_UserRequestIgnoresOuterText(t *testing.T) {
	raw := `<userRequest>
<environment_info>
The user's current OS is: Linux
</environment_info>
<userRequest>
你好啊，
</userRequest>   现在还是保存了这样的内容， 我希望的是只保存 你好啊
</userRequest>`

	got := BuildAuditPromptFromRawText(raw)

	require.Equal(t, "你好啊，", got)
}

func TestBuildAuditPromptForRequest_OpenAIResponsesInputStringWithoutUserTagsFallsBackToContent(t *testing.T) {
	req := &dto.OpenAIResponsesRequest{}
	wrappedJSON, err := common.Marshal(`<environment_context>internal context</environment_context>这是外层上下文`)
	require.NoError(t, err)
	req.Input = wrappedJSON

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "这是外层上下文", got)
}

func TestBuildAuditPromptForRequest_StripsRegexFragmentBeforeTaggedUserText(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: `(.*?)
<userRequest>明天是星期几</userRequest>`},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "明天是星期几", got)
}

func TestBuildAuditPromptForRequest_FallbackStripsSystemReminder(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: `<system-reminder>这些是系统提醒</system-reminder>明天是星期几`},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "明天是星期几", got)
}

func TestBuildAuditPromptForRequest_StripsRegexFragmentsAcrossMultipleTaggedMessages(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: `(.*?)
<userRequest>明天是星期几</userRequest>`},
			{Role: "user", Content: `(.*?)
<userRequest>昨夜雨疏风骤</userRequest>`},
		},
	}

	got := BuildAuditPromptForRequest(req)

	require.Equal(t, "明天是星期几\n昨夜雨疏风骤", got)
}

func TestNormalizeUserAuditText_StripsStandaloneRegexFragments(t *testing.T) {
	got := normalizeUserAuditText("(.*?)\n明天是星期几\n(.*?)")

	require.Equal(t, "明天是星期几", got)
}
