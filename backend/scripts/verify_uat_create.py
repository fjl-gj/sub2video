import argparse
import json
import os
import tempfile
import urllib.request
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
        description="Verify the deployed creation catalog and quote contract."
    )
    parser.add_argument(
        "--create-tasks",
        action="store_true",
        help="upload a test image and create text, first/last-frame, and multi-image tasks; this debits the UAT wallet and can invoke configured suppliers",
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

    def quote_for(input_type: str):
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
        st, quote = req("POST", "/client/tasks/quote", payload, token)
        forbidden = {"supplier", "cost", "upstream", "workflow", "adapter", "token"}
        leaked = [key for key in forbidden if key in json.dumps(quote).lower()]
        if leaked:
            raise RuntimeError(f"client quote leaked internal fields: {leaked}")
        return payload, quote

    text_params, quote = quote_for("text")
    _, keyframe_quote = quote_for("first_last")
    _, multi_quote = quote_for("multi_image")
    print("QUOTE_TEXT", st, quote["quoteId"], quote["price"]["total"])
    print("QUOTE_KEYFRAMES", keyframe_quote["quoteId"], keyframe_quote["price"]["total"])
    print("QUOTE_MULTI_IMAGE", multi_quote["quoteId"], multi_quote["price"]["total"])

    if not args.create_tasks:
        print("PREFLIGHT_OK no tasks created; rerun with --create-tasks for wallet, upload, and task acceptance")
        return

    st, created = req(
        "POST",
        "/client/tasks",
        {
            "quoteId": quote["quoteId"],
            "title": "wen-sheng-verify",
            "prompt": "rain street slow push",
        },
        token,
    )
    print("CREATE", st, created["task"]["id"], created["wallet"]["balance"])

    with tempfile.TemporaryDirectory(prefix="sub2video-uat-") as tmp:
        png = Path(tmp) / "test-upload.png"
        png.write_bytes(
            bytes.fromhex(
                "89504e470d0a1a0a0000000d4948445200000001000000010802000000907753de"
                "0000000c4944415408d763f8ffff3f0005fe02fe0c6e2f4d0000000049454e44ae426082"
            )
        )
        st, up = req(
            "POST",
            "/client/uploads",
            token=token,
            files=(str(png), "file", "test-upload.png", "image/png"),
        )
        print("UPLOAD", st, up["id"], up["sizeBytes"])

        st, created2 = req(
            "POST",
            "/client/tasks",
            {
                "quoteId": keyframe_quote["quoteId"],
                "title": "shou-wei-verify",
                "prompt": "camera push",
                "sourceMeta": {"first": {"type": "upload", "id": up["id"]}, "last": {"type": "upload", "id": up["id"]}},
            },
            token,
        )
        print("CREATE_KEYFRAMES", st, created2["task"]["id"], created2["wallet"]["balance"])

        st, created3 = req(
            "POST",
            "/client/tasks",
            {
                "quoteId": multi_quote["quoteId"],
                "title": "duo-tu-verify",
                "prompt": "consistent character walking through a rainy neon street",
                "sourceMeta": {"refs": [{"type": "upload", "id": up["id"]}]},
            },
            token,
        )
        print("CREATE_MULTI_IMAGE", st, created3["task"]["id"], created3["wallet"]["balance"])


if __name__ == "__main__":
    main()
