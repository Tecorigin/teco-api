#!/usr/bin/env python3
"""Exercise the knowledge-base APIs exposed through a new-api API Token."""

import argparse
import json
import os
import secrets
import sys
import time
from pathlib import Path
from typing import Any

import requests

TIMEOUT_SECONDS = 30
POLL_INTERVAL_SECONDS = 5


def print_response(response: requests.Response) -> None:
    print(f"<<< HTTP {response.status_code}")
    print(f"response body: {response.text or '<empty>'}")


def request_json(
    session: requests.Session,
    base_url: str,
    api_key: str,
    path: str,
    payload: dict[str, Any] | None = None,
) -> dict[str, Any] | None:
    print(f"\n>>> POST {base_url}{path}")
    if payload is not None:
        print(f"request body: {json.dumps(payload, ensure_ascii=False)}")
    response = session.post(
        f"{base_url}{path}",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        json=payload,
        timeout=TIMEOUT_SECONDS,
    )
    print_response(response)
    try:
        result = response.json()
    except json.JSONDecodeError:
        return None
    return result if response.ok and result.get("success") else None


def upload_document(
    session: requests.Session,
    base_url: str,
    api_key: str,
    kb_name: str,
    document_path: Path,
) -> dict[str, Any] | None:
    print(f"\n>>> POST {base_url}/v1/knowledge/upload_docs")
    print(f"upload: {document_path.name} ({document_path.stat().st_size} bytes)")
    with document_path.open("rb") as document:
        response = session.post(
            f"{base_url}/v1/knowledge/upload_docs",
            headers={"Authorization": f"Bearer {api_key}"},
            data={"kb_name": kb_name},
            files={"files": (document_path.name, document, "text/markdown")},
            timeout=TIMEOUT_SECONDS,
        )
    print_response(response)
    try:
        result = response.json()
    except json.JSONDecodeError:
        return None
    return result if response.ok and result.get("success") else None


def request_sse(
    session: requests.Session,
    base_url: str,
    api_key: str,
    path: str,
    payload: dict[str, Any],
) -> bool:
    print(f"\n>>> POST {base_url}{path} (SSE)")
    print(f"request body: {json.dumps(payload, ensure_ascii=False)}")
    with session.post(
        f"{base_url}{path}",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
        json=payload,
        stream=True,
        timeout=TIMEOUT_SECONDS,
    ) as response:
        print(f"<<< HTTP {response.status_code}")
        if not response.ok:
            print(f"response body: {response.text or '<empty>'}")
            return False
        received_content = False
        for line in response.iter_lines(decode_unicode=True):
            if not line or not line.startswith("data:"):
                continue
            print(f"SSE {line}")
            payload = line.removeprefix("data:").strip()
            if payload == "[DONE]":
                return received_content
            try:
                event = json.loads(payload)
            except json.JSONDecodeError:
                continue
            received_content = received_content or bool(event.get("content"))
    return received_content


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Test /v1/knowledge APIs with a new-api API Token"
    )
    parser.add_argument(
        "--url", required=True, help="new-api base URL, e.g. http://10.10.30.8:3002"
    )
    parser.add_argument("--api-key", required=True, help="new-api sk- API Key")
    parser.add_argument(
        "--document",
        default=str(Path(__file__).parent / "constant" / "README.md"),
        help="Markdown file to upload",
    )
    parser.add_argument(
        "--task-timeout",
        type=int,
        default=120,
        help="Upload polling timeout in seconds",
    )
    args = parser.parse_args()

    base_url = args.url.rstrip("/")
    document_path = Path(args.document)
    if not document_path.is_file():
        parser.error(f"document does not exist: {document_path}")

    session = requests.Session()
    kb_name = f"newapi_token_probe_{secrets.token_hex(6)}"
    session_id: str | None = None
    kb_created = False

    try:
        listing = request_json(session, base_url, args.api_key, "/v1/knowledge/list")
        if listing is None:
            print("FAIL: token cannot list its knowledge bases")
            return 1

        created = request_json(
            session,
            base_url,
            args.api_key,
            "/v1/knowledge/create",
            {"kb_name": kb_name, "description": "Temporary API Token integration test"},
        )
        if created is None:
            print("FAIL: token cannot create a knowledge base")
            return 1
        kb_created = True

        listed = request_json(session, base_url, args.api_key, "/v1/knowledge/list")
        if listed is None or not any(
            item.get("name") == kb_name for item in listed.get("data", [])
        ):
            print("FAIL: created knowledge base is not visible through the token")
            return 1

        upload = upload_document(
            session, base_url, args.api_key, kb_name, document_path
        )
        if upload is None or not upload.get("data", {}).get("task_id"):
            print("FAIL: token cannot start document upload")
            return 1

        task_id = upload["data"]["task_id"]
        deadline = time.monotonic() + args.task_timeout
        while time.monotonic() < deadline:
            task = request_json(
                session,
                base_url,
                args.api_key,
                "/v1/knowledge/task_status",
                {"task_id": task_id},
            )
            if task is None:
                print("FAIL: token cannot query upload task")
                return 1
            status = task.get("data", {}).get("status")
            if status == "completed":
                break
            if status == "failed":
                print(
                    f"FAIL: upstream upload task failed: {task.get('data', {}).get('message')}"
                )
                return 1
            time.sleep(POLL_INTERVAL_SECONDS)
        else:
            print("FAIL: upload task timed out")
            return 1

        docs = request_json(
            session,
            base_url,
            args.api_key,
            "/v1/knowledge/list_docs",
            {"kb_name": kb_name},
        )
        document_list = docs.get("data", []) if docs else []
        if not document_list:
            print("FAIL: token cannot list documents")
            return 1
        document_id = document_list[0].get("id")

        create_session = request_json(
            session, base_url, args.api_key, "/v1/knowledge/chat/create_session"
        )
        if create_session is None or not create_session.get("data", {}).get(
            "session_id"
        ):
            print("FAIL: token cannot create a chat session")
            return 1
        session_id = create_session["data"]["session_id"]

        sessions = request_json(
            session, base_url, args.api_key, "/v1/knowledge/chat/list_sessions"
        )
        if sessions is None or not any(
            item.get("session_id") == session_id for item in sessions.get("data", [])
        ):
            print("FAIL: token cannot list chat sessions")
            return 1

        chat_response = session.post(
            f"{base_url}/v1/knowledge/chat/completions",
            headers={
                "Authorization": f"Bearer {args.api_key}",
                "Content-Type": "application/json",
            },
            json={
                "knowledge_base": kb_name,
                "query": "请用一句话概括文档内容。",
                "stream": False,
                "return_docs": True,
            },
            timeout=TIMEOUT_SECONDS,
        )
        print("\n>>> POST " + f"{base_url}/v1/knowledge/chat/completions")
        print_response(chat_response)
        try:
            chat = chat_response.json()
        except json.JSONDecodeError:
            chat = None
        if not chat_response.ok or not chat or not chat.get("choices"):
            print("FAIL: token cannot call knowledge-base Q&A")
            return 1

        if not request_sse(
            session,
            base_url,
            args.api_key,
            "/v1/knowledge/chat/session_chat",
            {
                "query": "文档是什么类型？",
                "session_id": session_id,
                "stream": True,
                "return_docs": False,
            },
        ):
            print("FAIL: token cannot call raw SSE knowledge-base Q&A")
            return 1

        if document_id:
            deleted_docs = request_json(
                session,
                base_url,
                args.api_key,
                "/v1/knowledge/delete_docs",
                {"kb_name": kb_name, "doc_ids": [document_id]},
            )
            if deleted_docs is None:
                print("FAIL: token cannot delete documents")
                return 1

        cleared = request_json(
            session,
            base_url,
            args.api_key,
            "/v1/knowledge/clear",
            {"kb_name": kb_name},
        )
        if cleared is None:
            print("FAIL: token cannot clear a knowledge base")
            return 1

        print(
            "\nPASS: API Token can manage knowledge bases, documents, tasks, sessions, and both Q&A APIs."
        )
        return 0
    finally:
        if session_id:
            request_json(
                session,
                base_url,
                args.api_key,
                "/v1/knowledge/chat/delete_session",
                {"session_id": session_id},
            )
        if kb_created:
            request_json(
                session,
                base_url,
                args.api_key,
                "/v1/knowledge/delete",
                {"kb_name": kb_name},
            )


if __name__ == "__main__":
    sys.exit(main())
