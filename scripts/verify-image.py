#!/usr/bin/env python3
"""Exercise only a locally owned image/container and loopback synthetic state."""

import argparse
import hashlib
import http.client
import io
import json
import subprocess
import tarfile
import time
from pathlib import Path

from frontend_materials import verify_source  # pyright: ignore[reportMissingImports]


def docker(*args):
    return subprocess.check_output(["docker", *args]).decode().strip()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("image")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    output = Path(args.output)
    output.mkdir(parents=True, exist_ok=True)
    info = json.loads(docker("image", "inspect", args.image))[0]
    assert info["Config"]["User"] == "65532:65532", "runtime must remain non-root"
    for option in ["-L", "-version", "-buildconf"]:
        result = docker(
            "run",
            "--rm",
            "--network",
            "none",
            "--entrypoint",
            "/usr/bin/ffprobe",
            args.image,
            option,
        )
        (output / ("ffprobe" + option + ".txt")).write_text(result + "\n")
    container = docker(
        "run",
        "-d",
        "--rm",
        "--read-only",
        "--cap-drop=ALL",
        "--security-opt=no-new-privileges",
        "--tmpfs",
        "/config:uid=65532,gid=65532,mode=750",
        "--tmpfs",
        "/cache:uid=65532,gid=65532,mode=750",
        "-p",
        "127.0.0.1::8080",
        args.image,
    )
    try:
        port = docker("port", container, "8080/tcp").split(":")[-1]
        connection = http.client.HTTPConnection("127.0.0.1", int(port), timeout=60)

        def request(path, method="GET"):
            connection.request(method, path)
            response = connection.getresponse()
            if response.status != 200:
                response.read()
                raise OSError("owned fixture HTTP status " + str(response.status))
            return response

        for _ in range(60):
            try:
                with request("/readyz") as response:
                    assert response.status == 200
                    response.read()
                break
            except (OSError, TimeoutError):
                time.sleep(1)
        else:
            raise RuntimeError("owned container did not become ready")
        with request("/api/v1/system/info") as response:
            build = json.load(response)
        with request("/source") as response:
            assert response.headers["Content-Type"] == "application/gzip"
            assert "attachment;" in response.headers["Content-Disposition"]
            data = response.read()
            sha = hashlib.sha256(data).hexdigest()
            assert response.headers["ETag"] == '"' + sha + '"'
        (output / "source.tar.gz").write_bytes(data)
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:

            def read(name):
                file = archive.extractfile(name)
                assert file is not None
                return file.read()

            manifest = json.loads(read("MANIFEST.json"))
            assert (
                manifest["version"] == build["version"]
                and manifest["commit"] == build["commit"]
            )
            for name, entry in manifest["files"].items():
                content = read(name)
                assert (
                    len(content) == entry["bytes"]
                    and hashlib.sha256(content).hexdigest() == entry["sha256"]
                ), name
            names = archive.getnames()
            for name in names:
                assert not Path(name).is_absolute() and ".." not in Path(name).parts
                if name.startswith("mediagrap/"):
                    assert not any(
                        part
                        in {
                            ".git",
                            ".env",
                            ".env.local",
                            ".local",
                            ".vitest",
                            "runtime",
                            "__pycache__",
                        }
                        for part in Path(name).parts
                    ), name
            assert "mediagrap/LICENSE" in names
            assert "third-party/runtime/ffmpeg-7.1.1.tar.xz" in names
            assert "third-party/runtime/ffmpeg/link.map" in names
            assert "third-party/go/Go-LICENSE" in names
            assert "third-party/npm/packages.json" in names
            upstream = "third-party/frontend-upstream/"
            inventory = json.loads(read(upstream + "inventory.json"))
            assert inventory == json.loads(
                read("mediagrap/docs/legal/frontend-sources.json")
            )
            for source in inventory["sources"]:
                verify_source(read(upstream + source["archive"]), source)
            for notice in inventory["bundledNotices"]:
                assert (
                    hashlib.sha256(read(upstream + notice["notice"])).hexdigest()
                    == notice["sha256"]
                )
                bundle = read(
                    "third-party/npm/" + notice["bundledIn"] + "/" + notice["bundle"]
                )
                assert notice["sourceMarker"].encode() in bundle
            (output / "frontend-inventory.json").write_text(
                json.dumps(inventory, indent=2) + "\n"
            )
            (output / "MANIFEST.json").write_bytes(read("MANIFEST.json"))
        with request("/source", method="HEAD") as response:
            assert (
                int(response.headers["Content-Length"]) == len(data)
                and response.read() == b""
            )
        for path in ["/sources", "/sources?from=bookmark"]:
            with request(path) as response:
                assert response.headers["Content-Type"].startswith("text/html")
                assert b"<html" in response.read()
        for path in [
            "/source/../config",
            "/source/",
            "/source?path=/config",
            "/source?",
            "/%73ource",
        ]:
            connection.request("GET", path)
            with connection.getresponse() as response:
                assert response.status == 404, path
                response.read()
        report = {
            "build": build,
            "archiveSHA256": sha,
            "archiveBytes": len(data),
            "imageBytes": info["Size"],
            "imageID": info["Id"],
            "architecture": info["Architecture"],
            "user": info["Config"]["User"],
            "testOnly": manifest["testOnly"],
            "sourceGETAndHEAD": True,
            "sourcePathRejections": True,
            "legacySourcesBookmarks": True,
            "preferredFrontendSourcesAndNestedNotices": True,
        }
        (output / "verification.json").write_text(json.dumps(report, indent=2) + "\n")
        print(json.dumps(report, indent=2))
    finally:
        docker("stop", container)


if __name__ == "__main__":
    main()
