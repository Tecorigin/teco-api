#!/usr/bin/env bash
set -euo pipefail

BASE_URL="https://tecoapi.thuwayteczz.com/"
MODEL_NAME="Qwen3.8-27B"
API_KEY="sk-O5XHaGkOH2obeAl0NbJbnrMgxbjPt0Pccrq6FFh9myQQu8vu"

# BASE_URL="http://10.10.30.8:3000/"
# MODEL_NAME="Qwen3.8-27B"
# API_KEY="sk-wJrAcW1MTlBgPHJkS3E1zomyFmqtefj6hVsDQflAtmpgvKn7"

command -v jq >/dev/null || { printf 'jq is required\n' >&2; exit 1; }
command -v curl >/dev/null || { printf 'curl is required\n' >&2; exit 1; }

headers=(-H 'Content-Type: application/json' -H 'anthropic-version: 2023-06-01')
if [[ -n "$API_KEY" ]]; then
  headers+=(-H "Authorization: Bearer ${API_KEY}" -H "x-api-key: ${API_KEY}")
fi

printf 'POST %s/v1/messages (model=%s)\n' "${BASE_URL%/}" "$MODEL_NAME"

jq -n --arg model "$MODEL_NAME" '{
  model: $model,
  max_tokens: 128,
  stream: false,
  messages: [{role: "user", content: "Reply with OK. Do not call any tools."}],
  tools: [{
    name: "Agent",
    description: "Launch a new agent to handle complex, multi-step tasks.",
    input_schema: {
      type: "object",
      properties: {
        description: {type: "string", description: "A short (3-5 word) description of the task"},
        prompt: {type: "string", description: "The task for the agent to perform"},
        subagent_type: {type: "string"},
        model: {type: "string", enum: ["sonnet", "opus", "haiku", "fable"]},
        effort: {type: "string", enum: ["low", "medium", "high", "xhigh", "max"]},
        isolation: {type: "string", enum: ["worktree", "remote"]}
      },
      required: ["description", "prompt"],
      additionalProperties: false
    }
  }]
}' | curl --silent --show-error --include --fail-with-body \
  --connect-timeout 10 --max-time 120 \
  "${headers[@]}" \
  --data-binary @- \
  --write-out '\nHTTP status: %{http_code}\n' \
  "${BASE_URL%/}/v1/messages"【】