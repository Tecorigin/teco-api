#!/usr/bin/env python3
"""Diagnose a third-party knowledge base service independently of new-api.

Examples:
  KNOWLEDGE_BASE_URL=http://127.0.0.1:8500 \
  KNOWLEDGE_BASE_API_KEY=your-secret \
  python3 test_knowledge_base_service.py

  python3 test_knowledge_base_service.py \
    --url http://127.0.0.1:8500 --api-key your-secret --keep-resources
"""

import argparse
import hashlib
import hmac
import json
import os
import secrets
import sys
import time
import uuid
from dataclasses import dataclass
from typing import Any

import requests

REQUEST_TIMEOUT_SECONDS = 30
BODY_PREVIEW_LIMIT = 16 * 1024


@dataclass
class TestIdentity:
    org_id: str
    user_id: str
    org_name: str


class KnowledgeBaseProbe:
    def __init__(self, base_url: str, api_key: str) -> None:
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.session = requests.Session()
        self.failed = False

    def service_request(
        self,
        method: str,
        path: str,
        *,
        payload: dict[str, Any] | None = None,
        params: dict[str, str] | None = None,
    ) -> requests.Response:
        return self._request(
            method,
            path,
            headers={"X-Api-Key": self.api_key},
            payload=payload,
            params=params,
        )

    def user_request(
        self,
        method: str,
        path: str,
        identity: TestIdentity,
        *,
        payload: dict[str, Any] | None = None,
    ) -> requests.Response:
        timestamp = str(int(time.time()))
        nonce = uuid.uuid4().hex
        sign_string = ":".join(
            [identity.user_id, identity.org_id, "admin", timestamp, nonce]
        )
        signature = hmac.new(
            self.api_key.encode("utf-8"),
            sign_string.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        return self._request(
            method,
            path,
            headers={
                "X-User-Id": identity.user_id,
                "X-Org-Id": identity.org_id,
                "X-User-Role": "admin",
                "X-Timestamp": timestamp,
                "X-Nonce": nonce,
                "X-Signature": signature,
            },
            payload=payload,
        )

    def upload_document(
        self, identity: TestIdentity, kb_name: str, document_path: str
    ) -> requests.Response:
        timestamp = str(int(time.time()))
        nonce = uuid.uuid4().hex
        sign_string = ":".join(
            [identity.user_id, identity.org_id, "admin", timestamp, nonce]
        )
        signature = hmac.new(
            self.api_key.encode("utf-8"),
            sign_string.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        url = f"{self.base_url}/knowledge_base/upload_docs"
        headers = {
            "Accept": "application/json",
            "X-User-Id": identity.user_id,
            "X-Org-Id": identity.org_id,
            "X-User-Role": "admin",
            "X-Timestamp": timestamp,
            "X-Nonce": nonce,
            "X-Signature": signature,
        }
        file_size = os.path.getsize(document_path)
        print(f"\n>>> POST {url}")
        print(f"request headers: {format_headers(headers)}")
        print(
            f"request multipart: kb_name={kb_name!r}, "
            f"file={os.path.basename(document_path)!r} ({file_size} bytes)"
        )
        try:
            with open(document_path, "rb") as document:
                response = self.session.post(
                    url,
                    headers=headers,
                    data={"kb_name": kb_name},
                    files={
                        "files": (
                            os.path.basename(document_path),
                            document,
                            "text/markdown",
                        )
                    },
                    timeout=REQUEST_TIMEOUT_SECONDS,
                )
        except (OSError, requests.RequestException) as error:
            self.failed = True
            print(f"<<< request failed: {error}")
            raise
        self._print_response(response)
        return response

    def stream_chat(self, identity: TestIdentity, session_id: str, query: str) -> bool:
        timestamp = str(int(time.time()))
        nonce = uuid.uuid4().hex
        sign_string = ":".join(
            [identity.user_id, identity.org_id, "admin", timestamp, nonce]
        )
        signature = hmac.new(
            self.api_key.encode("utf-8"),
            sign_string.encode("utf-8"),
            hashlib.sha256,
        ).hexdigest()
        url = f"{self.base_url}/chat/session_chat"
        payload = {
            "query": query,
            "session_id": session_id,
            "stream": True,
            "return_docs": True,
        }
        headers = {
            "Accept": "text/event-stream",
            "Content-Type": "application/json",
            "X-User-Id": identity.user_id,
            "X-Org-Id": identity.org_id,
            "X-User-Role": "admin",
            "X-Timestamp": timestamp,
            "X-Nonce": nonce,
            "X-Signature": signature,
        }
        print(f"\n>>> POST {url} (SSE)")
        print(f"request headers: {format_headers(headers)}")
        print(f"request body: {json.dumps(payload, ensure_ascii=False)}")

        started_at = time.monotonic()
        try:
            with self.session.post(
                url,
                headers=headers,
                json=payload,
                stream=True,
                timeout=REQUEST_TIMEOUT_SECONDS,
            ) as response:
                print(f"<<< HTTP {response.status_code}")
                print(f"response headers: {format_headers(dict(response.headers))}")
                if not response.ok:
                    print(f"response body: {body_preview(response.text)}")
                    self.failed = True
                    return False

                event_count = 0
                received_content = False
                for line in response.iter_lines(decode_unicode=True):
                    if not line or not line.startswith("data:"):
                        continue
                    event_count += 1
                    elapsed = time.monotonic() - started_at
                    now = time.strftime("%Y-%m-%d %H:%M:%S")
                    data = line.removeprefix("data:").strip()
                    try:
                        event = json.loads(data)
                        content = event.get("content", "")
                        received_content = received_content or bool(content)
                        print(
                            f"[{now}] +{elapsed:.3f}s event={event_count} "
                            f"content={content!r}"
                        )
                    except json.JSONDecodeError:
                        print(
                            f"[{now}] +{elapsed:.3f}s event={event_count} data={data!r}"
                        )
                print(
                    f"SSE finished: events={event_count}, "
                    f"duration={time.monotonic() - started_at:.3f}s"
                )
                return received_content
        except requests.RequestException as error:
            self.failed = True
            print(f"<<< request failed: {error}")
            return False

    def _request(
        self,
        method: str,
        path: str,
        *,
        headers: dict[str, str],
        payload: dict[str, Any] | None = None,
        params: dict[str, str] | None = None,
    ) -> requests.Response:
        url = f"{self.base_url}{path}"
        request_headers = {"Accept": "application/json", **headers}
        if payload is not None:
            request_headers["Content-Type"] = "application/json"

        print(f"\n>>> {method} {url}")
        print(f"request headers: {format_headers(request_headers)}")
        if params:
            print(f"request query: {json.dumps(params, ensure_ascii=False)}")
        if payload is not None:
            print(f"request body: {json.dumps(payload, ensure_ascii=False)}")

        try:
            response = self.session.request(
                method,
                url,
                headers=request_headers,
                json=payload,
                params=params,
                timeout=REQUEST_TIMEOUT_SECONDS,
            )
        except requests.RequestException as error:
            self.failed = True
            print(f"<<< request failed: {error}")
            raise

        self._print_response(response)

        if not response.ok:
            self.failed = True
            return response

        try:
            envelope = response.json()
        except json.JSONDecodeError:
            self.failed = True
            print(
                "!!! expected a JSON response, but the service returned non-JSON content"
            )
            return response

        if envelope.get("code") != 200:
            self.failed = True
            print(
                "!!! API envelope reports failure: "
                f"code={envelope.get('code')!r}, msg={envelope.get('msg')!r}"
            )
        return response

    def _print_response(self, response: requests.Response) -> None:
        print(f"<<< HTTP {response.status_code}")
        print(f"response headers: {format_headers(dict(response.headers))}")
        print(f"response body: {body_preview(response.text)}")
        if not response.ok:
            self.failed = True
            return
        try:
            envelope = response.json()
        except json.JSONDecodeError:
            self.failed = True
            print(
                "!!! expected a JSON response, but the service returned non-JSON content"
            )
            return
        if envelope.get("code") != 200:
            self.failed = True
            print(
                "!!! API envelope reports failure: "
                f"code={envelope.get('code')!r}, msg={envelope.get('msg')!r}"
            )


def format_headers(headers: dict[str, str]) -> dict[str, str]:
    sensitive_names = {
        "authorization",
        "cookie",
        "set-cookie",
        "x-api-key",
        "x-nonce",
        "x-signature",
    }
    return {
        name: "***" if name.lower() in sensitive_names else value
        for name, value in headers.items()
    }


def body_preview(body: str) -> str:
    if len(body) <= BODY_PREVIEW_LIMIT:
        return body or "<empty>"
    return f"{body[:BODY_PREVIEW_LIMIT]}...<truncated>"


def response_data(response: requests.Response) -> Any | None:
    try:
        data = response.json()
    except json.JSONDecodeError:
        return None
    if response.ok and data.get("code") == 200:
        return data.get("data")
    return None


def successful(response: requests.Response) -> bool:
    try:
        return response.ok and response.json().get("code") == 200
    except json.JSONDecodeError:
        return False


def create_identity(args: argparse.Namespace) -> TestIdentity:
    suffix = secrets.token_hex(6)
    org_id = args.org_id or f"newapi_probe_org_{suffix}"
    user_id = args.user_id or f"newapi_probe_user_{suffix}"
    return TestIdentity(
        org_id=org_id,
        user_id=user_id,
        org_name=args.org_name or f"new-api knowledge-base probe {suffix}",
    )


def wait_for_upload_task(
    probe: KnowledgeBaseProbe,
    identity: TestIdentity,
    task_id: str,
    timeout_seconds: int,
) -> bool:
    deadline = time.monotonic() + timeout_seconds
    while time.monotonic() < deadline:
        response = probe.user_request(
            "POST",
            "/knowledge_base/task_status",
            identity,
            payload={"task_id": task_id},
        )
        task = response_data(response)
        if not successful(response) or not isinstance(task, dict):
            print("Stopping: task status query failed.")
            return False
        status = task.get("status")
        print(
            "task status: "
            f"status={status!r}, progress={task.get('progress')!r}, "
            f"message={task.get('message')!r}"
        )
        if status == "completed":
            return True
        if status == "failed":
            print(f"Upload task failed: {task.get('message')!r}")
            return False
        time.sleep(5)
    print(f"Upload task did not finish within {timeout_seconds} seconds.")
    return False


def run_probe(args: argparse.Namespace) -> bool:
    probe = KnowledgeBaseProbe(args.url, args.api_key)
    identity = create_identity(args)
    session_id: str | None = None
    kb_name = f"newapi_probe_kb_{secrets.token_hex(6)}"
    kb_created = False
    org_registered = False

    print("Knowledge base third-party service diagnostic")
    print(f"base URL: {probe.base_url}")
    print(f"test org ID: {identity.org_id}")
    print(f"test user ID: {identity.user_id}")
    print(f"test knowledge base: {kb_name}")

    try:
        register_response = probe.service_request(
            "POST",
            "/org/register",
            payload={"org_id": identity.org_id, "org_name": identity.org_name},
        )
        org_registered = successful(register_response)
        if not org_registered:
            print("Stopping: organization registration failed.")
            return False

        add_user_response = probe.service_request(
            "POST",
            "/user/add",
            payload={
                "user_id": identity.user_id,
                "org_id": identity.org_id,
                "role": "admin",
            },
        )
        if not successful(add_user_response):
            print("Stopping: administrator user registration failed.")
            return False

        org_info_response = probe.service_request(
            "GET", "/org/info", params={"org_id": identity.org_id}
        )
        if not successful(org_info_response):
            print("Stopping: organization lookup failed.")
            return False

        kb_list_response = probe.user_request("POST", "/knowledge_base/list", identity)
        if not successful(kb_list_response):
            print(
                "Knowledge base list failed. This is the same endpoint new-api calls."
            )
            return False

        create_kb_response = probe.user_request(
            "POST",
            "/knowledge_base/create",
            identity,
            payload={
                "kb_name": kb_name,
                "description": "Temporary new-api integration diagnostic knowledge base",
            },
        )
        kb_created = successful(create_kb_response)
        if not kb_created:
            print("Stopping: knowledge base creation failed.")
            return False

        list_after_create_response = probe.user_request(
            "POST", "/knowledge_base/list", identity
        )
        listed_kbs = response_data(list_after_create_response)
        if not successful(list_after_create_response) or not any(
            kb.get("name") == kb_name for kb in (listed_kbs or [])
        ):
            print("Stopping: created knowledge base was not returned by list.")
            return False

        list_docs_response = probe.user_request(
            "POST", "/knowledge_base/list_docs", identity, payload={"kb_name": kb_name}
        )
        if not successful(list_docs_response):
            print("Stopping: knowledge base document list failed.")
            return False

        upload_response = probe.upload_document(identity, kb_name, args.document)
        upload_data = response_data(upload_response)
        if not successful(upload_response) or not isinstance(upload_data, dict):
            print("Stopping: document upload task creation failed.")
            return False
        task_id = upload_data.get("task_id")
        if not task_id:
            print("Stopping: upload response did not include task_id.")
            return False
        if not wait_for_upload_task(probe, identity, str(task_id), args.task_timeout):
            return False

        list_docs_after_upload_response = probe.user_request(
            "POST", "/knowledge_base/list_docs", identity, payload={"kb_name": kb_name}
        )
        documents = response_data(list_docs_after_upload_response)
        if not successful(list_docs_after_upload_response) or not any(
            document.get("name") == os.path.basename(args.document)
            and document.get("run") == "DONE"
            for document in (documents or [])
        ):
            print("Stopping: uploaded Markdown document was not parsed with run=DONE.")
            return False

        create_session_response = probe.user_request(
            "POST", "/chat/create_session", identity
        )
        session_data = response_data(create_session_response)
        if session_data is None or not session_data.get("session_id"):
            print("Stopping: chat session creation failed.")
            return False
        session_id = str(session_data["session_id"])
        if not probe.stream_chat(identity, session_id, args.chat_query):
            print("Stopping: chat stream returned no content.")
            return False

        clear_kb_response = probe.user_request(
            "POST", "/knowledge_base/clear", identity, payload={"kb_name": kb_name}
        )
        if not successful(clear_kb_response):
            print("Stopping: knowledge base clear failed.")
            return False

        delete_kb_response = probe.user_request(
            "POST", "/knowledge_base/delete", identity, payload={"kb_name": kb_name}
        )
        kb_created = not successful(delete_kb_response)
        if kb_created:
            print("Stopping: knowledge base deletion failed.")
            return False

        list_after_delete_response = probe.user_request(
            "POST", "/knowledge_base/list", identity
        )
        listed_kbs = response_data(list_after_delete_response)
        if not successful(list_after_delete_response) or any(
            kb.get("name") == kb_name for kb in (listed_kbs or [])
        ):
            print("Stopping: deleted knowledge base is still returned by list.")
            return False

        list_sessions_response = probe.user_request(
            "POST", "/chat/list_sessions", identity
        )
        if not successful(list_sessions_response):
            print("Chat session list failed.")
            return False

        print(
            "\nPASS: organization, user, knowledge-base CRUD, and chat session endpoints succeeded."
        )
        return not probe.failed
    except requests.RequestException:
        return False
    finally:
        if session_id:
            try:
                probe.user_request(
                    "POST",
                    "/chat/delete_session",
                    identity,
                    payload={"session_id": session_id},
                )
            except requests.RequestException as error:
                print(
                    f"Cleanup warning: failed to delete session {session_id}: {error}"
                )
        if kb_created:
            try:
                probe.user_request(
                    "POST",
                    "/knowledge_base/delete",
                    identity,
                    payload={"kb_name": kb_name},
                )
            except requests.RequestException as error:
                print(
                    f"Cleanup warning: failed to delete knowledge base {kb_name}: {error}"
                )
        if org_registered and not args.keep_resources:
            try:
                probe.service_request(
                    "POST", "/org/unregister", payload={"org_id": identity.org_id}
                )
            except requests.RequestException as error:
                print(
                    f"Cleanup warning: failed to unregister organization {identity.org_id}: {error}"
                )
        elif org_registered:
            print(f"\nKeeping test organization: {identity.org_id}")


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Directly test a third-party knowledge base service used by new-api."
    )
    parser.add_argument(
        "--url",
        default=os.environ.get("KNOWLEDGE_BASE_URL"),
        help="Third-party base URL (default: KNOWLEDGE_BASE_URL)",
    )
    parser.add_argument(
        "--api-key",
        default=os.environ.get("KNOWLEDGE_BASE_API_KEY"),
        help="Shared secret (default: KNOWLEDGE_BASE_API_KEY)",
    )
    parser.add_argument(
        "--org-id", help="Use an existing or explicit test organization ID"
    )
    parser.add_argument(
        "--org-name", help="Organization name for a generated test organization"
    )
    parser.add_argument("--user-id", help="Use an explicit test user ID")
    parser.add_argument(
        "--document",
        default=os.path.join(os.path.dirname(__file__), "constant", "README.md"),
        help="Markdown file to upload (default: constant/README.md)",
    )
    parser.add_argument(
        "--task-timeout",
        type=int,
        default=300,
        help="Maximum seconds to wait for document parsing (default: 300)",
    )
    parser.add_argument(
        "--chat-query",
        default="constant 包只能被其他包引用吗？",
        help="Question sent to the uploaded Markdown document (default: an exact document topic)",
    )
    parser.add_argument(
        "--keep-resources",
        action="store_true",
        help="Do not unregister the temporary organization after testing",
    )
    args = parser.parse_args()

    if not args.url:
        parser.error("provide --url or set KNOWLEDGE_BASE_URL")
    if not args.api_key:
        parser.error("provide --api-key or set KNOWLEDGE_BASE_API_KEY")
    if not os.path.isfile(args.document):
        parser.error(f"Markdown test document does not exist: {args.document}")
    if not args.document.lower().endswith((".md", ".markdown")):
        parser.error("--document must point to a Markdown file")
    if args.task_timeout < 5:
        parser.error("--task-timeout must be at least 5 seconds")

    return 0 if run_probe(args) else 1


if __name__ == "__main__":
    sys.exit(main())
