"""The export runner's object storage calls, over the S3 API with boto3.

Credentials, the endpoint and the region come from the environment
(AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, S3_ENDPOINT, S3_REGION); nothing
here prints them. Each command fails loudly, with a one-line reason on
stderr and a non-zero exit.

    s3.py servers BUCKET PREFIX        the backup servers under PREFIX, one a line
    s3.py download BUCKET DIR SKIP     every object but those under SKIP into DIR;
                                       prints how many
    s3.py put-stream BUCKET KEY        stdin to KEY; prints {"size", "sha256"}
    s3.py put-json BUCKET KEY JSON     a small JSON document to KEY
    s3.py size BUCKET KEY              the object's size in bytes
"""

import hashlib
import json
import os
import sys

import boto3
from boto3.s3.transfer import TransferConfig
from botocore.config import Config

# Parts of 128 MiB, two at a time: an archive of up to 1.28 TB (S3's 10,000
# parts) in about 256 MiB of memory.
TRANSFER = TransferConfig(multipart_threshold=128 << 20, multipart_chunksize=128 << 20, max_concurrency=2)


def client():
    return boto3.client(
        "s3",
        endpoint_url=os.environ["S3_ENDPOINT"],
        region_name=os.environ["S3_REGION"],
        config=Config(s3={"addressing_style": "path"}, retries={"max_attempts": 10, "mode": "standard"}),
    )


def servers(bucket, prefix):
    pages = client().get_paginator("list_objects_v2").paginate(Bucket=bucket, Prefix=prefix, Delimiter="/")
    for page in pages:
        for p in page.get("CommonPrefixes", []):
            name = p["Prefix"][len(prefix):].rstrip("/")
            if name:
                print(name)


def safe_path(root, key):
    """Where an object is written: inside root, never above it."""
    if key.startswith("/") or any(part in ("", ".", "..") for part in key.split("/")):
        raise SystemExit(f"refusing the object key {key!r}: it is not a plain relative path")
    path = os.path.normpath(os.path.join(root, key))
    if not path.startswith(os.path.normpath(root) + os.sep):
        raise SystemExit(f"refusing the object key {key!r}: it leaves the documents folder")
    return path


def download(bucket, root, skip):
    c = client()
    count = 0
    os.makedirs(root, exist_ok=True)
    for page in c.get_paginator("list_objects_v2").paginate(Bucket=bucket):
        for obj in page.get("Contents", []):
            key = obj["Key"]
            if key.endswith("/") or (skip and key.startswith(skip)):
                continue
            path = safe_path(root, key)
            os.makedirs(os.path.dirname(path), exist_ok=True)
            c.download_file(bucket, key, path, Config=TRANSFER)
            count += 1
    print(count)


class Hashing:
    """Reads a stream once, counting and hashing what passes."""

    def __init__(self, stream):
        self.stream = stream
        self.size = 0
        self.sha256 = hashlib.sha256()

    def read(self, n=-1):
        chunk = self.stream.read(n)
        self.size += len(chunk)
        self.sha256.update(chunk)
        return chunk


def put_stream(bucket, key):
    body = Hashing(sys.stdin.buffer)
    client().upload_fileobj(body, bucket, key, Config=TRANSFER, ExtraArgs={"ContentType": "application/x-tar"})
    if body.size == 0:
        raise SystemExit("the archive was empty")
    print(json.dumps({"size": body.size, "sha256": body.sha256.hexdigest()}))


def put_json(bucket, key, document):
    json.loads(document)
    client().put_object(Bucket=bucket, Key=key, Body=document.encode(), ContentType="application/json")


def size(bucket, key):
    print(client().head_object(Bucket=bucket, Key=key)["ContentLength"])


COMMANDS = {
    "servers": (servers, 2),
    "download": (download, 3),
    "put-stream": (put_stream, 2),
    "put-json": (put_json, 3),
    "size": (size, 2),
}


def main(argv):
    if len(argv) < 2 or argv[1] not in COMMANDS or len(argv) - 2 != COMMANDS[argv[1]][1]:
        raise SystemExit(__doc__)
    fn, _ = COMMANDS[argv[1]]
    try:
        fn(*argv[2:])
    except SystemExit:
        raise
    except Exception as e:  # noqa: BLE001 - one line for the runner's log, never a traceback with request details
        raise SystemExit(f"s3.py {argv[1]}: {type(e).__name__}: {e}") from None


if __name__ == "__main__":
    main(sys.argv)
