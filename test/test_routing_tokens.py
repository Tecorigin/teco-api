"""Generate a token-sized prompt and send one chat completion request."""

import argparse
import json
import sys
import time
import urllib.error
import urllib.request

URL = "http://10.10.30.8:3002"
MODEL = "Qwen3.8-27B"
API_KEY = "sk-oJqVTrDu5QzYFhcGJR3Dteb0M5JbDp8CyzJ7pN3ePrLdKCZr"
ENCODING = "cl100k_base"
TIMEOUT = 300


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--tokens", required=True, type=int, help="Prompt text tokens, excluding message overhead")
    args = parser.parse_args()
    if not 1 <= args.tokens <= 1000000:
        parser.error("--tokens must be between 1 and 1000000")

    try:
        import tiktoken
    except ImportError:
        print("Install dependency: pip install tiktoken", file=sys.stderr)
        return 1

    try:
        encoder = tiktoken.get_encoding(ENCODING)
    except ValueError as error:
        print(str(error), file=sys.stderr)
        return 1

    prompt = " hello" * args.tokens
    actual_tokens = len(encoder.encode(prompt))
    if actual_tokens != args.tokens:
        print(f"Encoding {ENCODING} produced {actual_tokens} tokens instead of {args.tokens}; no request sent.", file=sys.stderr)
        return 1

    if not API_KEY.strip():
        print("Set API_KEY at the top of this script first.", file=sys.stderr)
        return 1

    root = URL.rstrip("/")
    if root.endswith("/v1/chat/completions"):
        endpoint = root
    elif root.endswith("/v1"):
        endpoint = root + "/chat/completions"
    else:
        endpoint = root + "/v1/chat/completions"

    payload = {
        "model": MODEL,
        "messages": [{"role": "user", "content": prompt}],
        "stream": False,
        "max_tokens": 16,
    }
    request = urllib.request.Request(
        endpoint,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json", "Authorization": f"Bearer {API_KEY.strip()}"},
        method="POST",
    )
    print(f"URL: {endpoint}")
    print(f"Model: {MODEL}")
    print(f"Prompt tokens: {actual_tokens} ({ENCODING}; message overhead excluded)")
    print("Sending one request. Check the platform usage logs for the selected channel.")
    started = time.perf_counter()
    try:
        response = urllib.request.urlopen(request, timeout=TIMEOUT)
    except urllib.error.HTTPError as error:
        response = error
    except (urllib.error.URLError, TimeoutError) as error:
        print(f"Request failed: {error}", file=sys.stderr)
        return 1

    with response:
        body = response.read().decode("utf-8", errors="replace")
        print(f"HTTP status: {response.code}")
        print(f"Elapsed: {time.perf_counter() - started:.3f}s")
        for name in ("X-Request-Id", "X-Teco-Routing-Version", "X-Teco-Routing-Class", "X-Teco-Routing-Count-Mode"):
            if response.headers.get(name):
                print(f"{name}: {response.headers[name]}")
        try:
            result = json.loads(body)
            print(json.dumps(result, ensure_ascii=False, indent=2))
        except json.JSONDecodeError:
            print(body[:4000])
        return 0 if response.code < 400 else 1


if __name__ == "__main__":
    sys.exit(main())