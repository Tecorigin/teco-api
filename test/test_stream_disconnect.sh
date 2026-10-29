#!/usr/bin/env bash

auth_header=$(sed -n 's/.*-H "\(Authorization: Bearer [^"]*\)".*/\1/p' "$(dirname "$0")/test_curl.sh")

curl -sS -N -i --max-time 60 http://localhost:3000/v1/chat/completions \
  -H "$auth_header" \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "MiniMax-M2.5-AWQ",
    "messages": [{"role": "user", "content": "Write a very long, detailed tutorial on building an operating system from scratch. Include extensive code examples for every step. Keep writing until the output limit is reached."}],
    "stream": true,
    "max_tokens": 4096
  }' | head -n 60

printf '\nConnection cut after 60 header/body lines. Check the printed request ID in usage logs.\n'