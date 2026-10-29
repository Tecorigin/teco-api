python3 - <<'PY' | curl http://10.10.30.8:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-wJrAcW1MTlBgPHJkS3E1zomyFmqtefj6hVsDQflAtmpgvKn7" \
  -H "Content-Type: application/json" \
  --data-binary @-
import json

input_tokens = 1000   # 1000 / 10000 / 20000
output_tokens = 1000 # 1000 / 10000 / 20000

print(json.dumps({
    "model": "Qwen3.8-27B",
    "messages": [{
        "role": "user11",
        "content": "Explain quantum entanglement. Padding: " + "test " * input_tokens
    }],
    "temperature": 0.7,
    "max_tokens": output_tokens
}))
PY