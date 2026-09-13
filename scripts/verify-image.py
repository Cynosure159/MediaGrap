#!/usr/bin/env python3
"""Exercise only a locally owned image/container and loopback synthetic state."""

import argparse
import functools
import hashlib
import http.client
import http.server
import io
import json
import subprocess
import tarfile
import threading
import time
from pathlib import Path

from source_contract import download_public, source_identity, verify_archive


def docker(*args):
    return subprocess.check_output(["docker", *args]).decode().strip()


def file_digest(filesystem, name):
    stream = filesystem.extractfile(name)
    if stream is None:
        raise ValueError("missing image component: " + name)
    return hashlib.sha256(stream.read()).hexdigest()


def verify_image_filesystem(exported, data, manifest):
    with tarfile.open(fileobj=io.BytesIO(exported)) as filesystem:
        names = filesystem.getnames()
        assert not any(
            name.endswith((".tar.gz", ".tar.xz", ".tar.bz2", ".zip")) for name in names
        ), "source archive retained in runtime"
        binary = filesystem.getmember("mediagrap")
        assert binary.size < len(data), "source-sized application binary"
        for name, entry in manifest["files"].items():
            if name.startswith(
                ("third-party/runtime/notices/", "third-party/runtime/ffmpeg/")
            ):
                runtime_name = name.replace(
                    "third-party/runtime/", "usr/share/mediagrap/", 1
                )
            elif name == "mediagrap/LICENSE":
                runtime_name = "usr/share/mediagrap/LICENSE"
            else:
                continue
            stream = filesystem.extractfile(runtime_name)
            assert (
                stream is not None
                and hashlib.sha256(stream.read()).hexdigest() == entry["sha256"]
            ), runtime_name
        with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as source:
            stream = source.extractfile("third-party/runtime/runtime-sha256.txt")
            assert stream is not None
            for line in stream.read().decode().splitlines():
                expected, path = line.split()
                stream = filesystem.extractfile(path.removeprefix("/out").lstrip("/"))
                assert (
                    stream is not None
                    and hashlib.sha256(stream.read()).hexdigest() == expected
                ), path
        return {
            "binaryBytes": binary.size,
            "imageCASHA256": file_digest(
                filesystem, "etc/ssl/certs/ca-certificates.crt"
            ),
        }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("image")
    parser.add_argument("--output", required=True)
    parser.add_argument(
        "--source-directory",
        help="Owned build artifacts: serve via loopback fixture instead of public GitHub",
    )
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
            info["Id"],
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
        info["Id"],
    )
    try:
        port = docker("port", container, "8080/tcp").split(":")[-1]
        connection = http.client.HTTPConnection("127.0.0.1", int(port), timeout=60)

        def request(path, method="GET", status=200):
            connection.request(method, path)
            response = connection.getresponse()
            if response.status != status:
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
        arch = info["Architecture"]
        _, asset, url = source_identity(build["version"], build["commit"], arch)
        assert build["sourceURL"] == url and build["sourceArchitecture"] == arch
        sha = build["sourceSHA256"]
        for method in ("GET", "HEAD"):
            with request("/source", method=method, status=307) as response:
                assert response.headers["Location"] == url
                assert response.headers["X-Source-SHA256"] == sha
                body = response.read()
                if method == "HEAD":
                    assert body == b""
        if args.source_directory:
            directory = Path(args.source_directory).resolve()
            metadata = json.loads((directory / "manifest.json").read_text())
            assert (
                metadata["url"] == url
                and metadata["sha256"] == sha
                and metadata["asset"] == asset
            )
            # This mapping is verifier-only: the binary still advertises the pinned HTTPS URL.
            handler = functools.partial(
                http.server.SimpleHTTPRequestHandler, directory=str(directory)
            )
            fixture = http.server.ThreadingHTTPServer(("127.0.0.1", 0), handler)
            thread = threading.Thread(target=fixture.serve_forever, daemon=True)
            thread.start()
            try:
                fixture_connection = http.client.HTTPConnection(
                    "127.0.0.1", fixture.server_port, timeout=120
                )
                fixture_connection.request("GET", "/source.tar.gz")
                response = fixture_connection.getresponse()
                assert response.status == 200
                data = response.read()
                fixture_connection.close()
            finally:
                fixture.shutdown()
                fixture.server_close()
                thread.join()
        else:
            data = download_public(url)
        manifest, inventory = verify_archive(data, {**build, "architecture": arch}, sha)
        exported = subprocess.check_output(["docker", "export", container])
        # Verify distributed bytes before starting: hosts can alter trust stores at startup.
        image_container = docker("create", info["Id"])
        try:
            details = json.loads(docker("inspect", image_container))[0]
            assert details["Image"] == info["Id"] and not details["State"]["Running"]
            assert all(
                mount["Destination"] in ("/config", "/cache")
                for mount in details["Mounts"]
            )
            image_export = subprocess.check_output(
                ["docker", "export", image_container]
            )
        finally:
            docker("rm", "-v", image_container)
        filesystem_proof = verify_image_filesystem(image_export, data, manifest)
        with tarfile.open(fileobj=io.BytesIO(exported)) as started:
            started_ca = file_digest(started, "etc/ssl/certs/ca-certificates.crt")
        (output / "source.tar.gz").write_bytes(data)
        (output / "MANIFEST.json").write_text(json.dumps(manifest, indent=2) + "\n")
        (output / "frontend-inventory.json").write_text(
            json.dumps(inventory, indent=2) + "\n"
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
            "sourceVerification": "local-fixture"
            if args.source_directory
            else "public-unauthenticated",
            "archiveSHA256": sha,
            "archiveBytes": len(data),
            "imageBytes": info["Size"],
            **filesystem_proof,
            "startedContainerID": container,
            "imageFilesystemContainerID": image_container,
            "dockerServer": json.loads(
                docker("version", "--format", "{{json .Server}}")
            ),
            "startedCASHA256": started_ca,
            "startedCADiffersFromImage": started_ca
            != filesystem_proof["imageCASHA256"],
            "runtimeSourceArchivesAbsent": True,
            "runtimeNoticesAndComponentsMatchSource": True,
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
