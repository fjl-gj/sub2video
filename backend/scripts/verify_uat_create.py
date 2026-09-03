import json
import urllib.request
from pathlib import Path

base = "http://192.168.0.107:12000/api/v1"


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
    st, login = req(
        "POST",
        "/auth/login",
        {"email": "demo@sub2video.local", "password": "Demo1234!"},
    )
    print("LOGIN", st, login["user"]["balance"], login["user"]["id"])
    token = login["token"]["accessToken"]

    st, quote = req(
        "POST",
        "/client/tasks/quote",
        {"workflowId": "wf-h3-text", "durationSeconds": 5, "resolution": "768P"},
        token,
    )
    print("QUOTE", st, quote["estimatedCost"])
    st, created = req(
        "POST",
        "/client/tasks",
        {
            "workflowId": "wf-h3-text",
            "title": "wen-sheng-verify",
            "prompt": "rain street slow push",
            "aspectRatio": "16:9",
            "durationSeconds": 5,
            "resolution": "768P",
            "source": {"type": "text"},
            "priceSnapshot": {
                **quote["priceSnapshot"],
                "estimatedCost": quote["estimatedCost"],
                "pricePerSecond": quote["pricePerSecond"],
            },
        },
        token,
    )
    print("CREATE", st, created["task"]["id"], created["wallet"]["balance"])

    png = Path(r"E:\apps\github.com\sbu2video\backend\data\test-upload.png")
    png.parent.mkdir(parents=True, exist_ok=True)
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

    st, q2 = req(
        "POST",
        "/client/tasks/quote",
        {"workflowId": "wf-h3-image", "durationSeconds": 5, "resolution": "768P"},
        token,
    )
    st, created2 = req(
        "POST",
        "/client/tasks",
        {
            "workflowId": "wf-h3-image",
            "title": "tu-sheng-verify",
            "prompt": "camera push",
            "aspectRatio": "9:16",
            "durationSeconds": 5,
            "resolution": "768P",
            "source": {"type": "upload", "uploadId": up["id"]},
            "priceSnapshot": {
                **q2["priceSnapshot"],
                "estimatedCost": q2["estimatedCost"],
                "pricePerSecond": q2["pricePerSecond"],
            },
        },
        token,
    )
    print("CREATE_IMG", st, created2["task"]["id"], created2["wallet"]["balance"])


if __name__ == "__main__":
    main()
