import argparse
import json
import os
import time
import urllib.request
import uuid
from pathlib import Path

base = os.environ.get("SUB2VIDEO_UAT_BASE", "http://127.0.0.1:12000/api/v1").rstrip("/")


def req(method, path, data=None, token=None, files=None):
    url = base + path
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if files is None:
        body = None
        if data is not None:
            body = json.dumps(data).encode()
            headers["Content-Type"] = "application/json"
        request = urllib.request.Request(url, data=body, headers=headers, method=method)
        with urllib.request.urlopen(request, timeout=30) as resp:
            return resp.status, json.loads(resp.read().decode())

    boundary = "----s2vboundary"
    file_path, field, filename, ctype = files
    raw = Path(file_path).read_bytes()
    parts = [
        (
            f"--{boundary}\r\n"
            f'Content-Disposition: form-data; name="{field}"; filename="{filename}"\r\n'
            f"Content-Type: {ctype}\r\n\r\n"
        ).encode(),
        raw,
        f"\r\n--{boundary}--\r\n".encode(),
    ]
    body = b"".join(parts)
    headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"
    request = urllib.request.Request(url, data=body, headers=headers, method="POST")
    with urllib.request.urlopen(request, timeout=60) as resp:
        return resp.status, json.loads(resp.read().decode())


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Verify the deployed creation catalog, estimate contract, and optional task delivery."
    )
    parser.add_argument(
        "--create-tasks",
        action="store_true",
        help="create one text-to-video task; this debits the UAT wallet and invokes the configured supplier",
    )
    parser.add_argument(
        "--wait-seconds",
        type=int,
        default=300,
        help="maximum time to wait for the created task to reach persisted delivery (default: 300)",
    )
    args = parser.parse_args()

    st, login = req(
        "POST",
        "/auth/login",
        {"email": "demo@sub2video.local", "password": "Demo1234!"},
    )
    print("LOGIN", st, login["user"]["balance"], login["user"]["id"])
    token = login["token"]["accessToken"]

    st, catalog = req("GET", "/client/create/catalog", token=token)
    modes = {item["inputType"]: item for item in catalog["modes"] if item["enabled"]}
    models = catalog["models"]

    def estimate_for(input_type: str):
        mode = modes.get(input_type)
        if mode is None:
            raise RuntimeError(f"catalog does not expose the required {input_type} mode")
        model = next((item for item in models if input_type in item["inputTypes"]), None)
        if model is None:
            raise RuntimeError(f"catalog has no executable model for {input_type}")
        st, options = req("GET", f"/client/create/options?modeId={mode['id']}&modelId={model['id']}", token=token)
        payload = {
            "modeId": mode["id"],
            "modelId": model["id"],
            "durationSeconds": options["durations"][0],
            "resolution": options["resolutions"][0],
            "aspectRatio": options["aspectRatios"][0],
            "generateAudio": False,
        }
        st, estimate = req("POST", "/client/tasks/estimate", payload, token)
        forbidden = {"supplier", "cost", "upstream", "workflow", "adapter", "token"}
        leaked = [key for key in forbidden if key in json.dumps(estimate).lower()]
        if leaked:
            raise RuntimeError(f"client estimate leaked internal fields: {leaked}")
        if not estimate.get("available") or estimate.get("billing", {}).get("estimatedTotal", 0) <= 0:
            raise RuntimeError(f"estimate unavailable for {input_type}: {estimate}")
        if "quoteId" in estimate:
            raise RuntimeError("estimate response must not contain quoteId")
        return payload, estimate

    text_params, text_estimate = estimate_for("text")
    _, keyframe_estimate = estimate_for("first_last")
    _, multi_estimate = estimate_for("multi_image")
    print("ESTIMATE_TEXT", text_estimate["billing"]["estimatedTotal"])
    print("ESTIMATE_KEYFRAMES", keyframe_estimate["billing"]["estimatedTotal"])
    print("ESTIMATE_MULTI_IMAGE", multi_estimate["billing"]["estimatedTotal"])

    if not args.create_tasks:
        print("PREFLIGHT_OK no tasks created; rerun with --create-tasks for one charged task and delivery check")
        return

    request_id = f"uat-{uuid.uuid4().hex}"
    create_payload = {
        **text_params,
        "clientRequestId": request_id,
        "title": "wen-sheng-verify",
        "prompt": "a slow camera push along a quiet rainy street, natural light",
    }
    st, created = req(
        "POST",
        "/client/tasks",
        create_payload,
        token,
    )
    task_id = created["task"]["id"]
    print("CREATE", st, task_id, created["wallet"]["balance"])

    deadline = time.time() + max(args.wait_seconds, 1)
    while time.time() < deadline:
        st, detail = req("GET", f"/client/tasks/{task_id}", token=token)
        status = detail.get("status")
        delivery = detail.get("deliveryStatus")
        print("POLL", status, delivery)
        if status == "失败":
            raise RuntimeError(f"task failed: {detail.get('failureReason', 'unknown')}")
        if status == "已完成" and delivery == "ready" and detail.get("downloadUrl"):
            print("DELIVERY_OK", task_id, detail["downloadUrl"])
            return
        time.sleep(10)
    raise RuntimeError(f"task {task_id} did not reach persisted delivery within {args.wait_seconds}s")


if __name__ == "__main__":
    main()
