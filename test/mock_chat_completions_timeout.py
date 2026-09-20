#!/usr/bin/env python3
"""OpenAI-compatible mock server that delays non-streaming chat completions."""

import argparse
import json
import time
import uuid
from http import HTTPStatus
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

DEFAULT_DELAY_SECONDS = 8 * 60


class ChatCompletionsHandler(BaseHTTPRequestHandler):
    delay_seconds = DEFAULT_DELAY_SECONDS

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self.send_error(
                HTTPStatus.NOT_FOUND, "only /v1/chat/completions is supported"
            )
            return

        content_length = int(self.headers.get("Content-Length", 0))
        request_body = self.rfile.read(content_length)
        try:
            request = json.loads(request_body) if request_body else {}
        except json.JSONDecodeError:
            self.send_error(HTTPStatus.BAD_REQUEST, "request body must be valid JSON")
            return

        request_id = uuid.uuid4().hex
        print(
            f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] "
            f"received request {request_id}; waiting {self.delay_seconds}s",
            flush=True,
        )
        time.sleep(self.delay_seconds)

        response = {
            "id": f"chatcmpl-{request_id}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": request.get("model", "mock-model"),
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": "This is a delayed mock response.",
                    },
                    "finish_reason": "stop",
                }
            ],
            "usage": {
                "prompt_tokens": 1,
                "completion_tokens": 7,
                "total_tokens": 8,
            },
        }
        response_body = json.dumps(response).encode("utf-8")

        try:
            self.send_response(HTTPStatus.OK)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(response_body)))
            self.end_headers()
            self.wfile.write(response_body)
            print(
                f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] replied to {request_id}",
                flush=True,
            )
        except BrokenPipeError:
            print(
                f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] client disconnected: {request_id}",
                flush=True,
            )

    def log_message(self, format, *args):
        return


def main():
    parser = argparse.ArgumentParser(
        description="Delayed OpenAI Chat Completions mock server"
    )
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=18080)
    parser.add_argument("--delay-seconds", type=int, default=DEFAULT_DELAY_SECONDS)
    args = parser.parse_args()

    if args.delay_seconds < 0:
        parser.error("--delay-seconds must be non-negative")

    ChatCompletionsHandler.delay_seconds = args.delay_seconds
    server = ThreadingHTTPServer((args.host, args.port), ChatCompletionsHandler)
    print(
        f"listening on http://{args.host}:{args.port}/v1/chat/completions", flush=True
    )
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        server.server_close()


if __name__ == "__main__":
    main()
