"""Pinned public source identity shared by build, verification and publication."""

import hashlib
import io
import json
import re
import tarfile
import urllib.request
from pathlib import PurePosixPath
from urllib.parse import urlsplit

from frontend_materials import verify_source

REPOSITORY = "Cynosure159/MediaGrap"
STABLE = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
RC = re.compile(STABLE.pattern + r"-rc\.[1-9][0-9]*")
RELEASE_TAG = re.compile(r"v(?:" + STABLE.pattern + r"|" + RC.pattern + r")")


def source_identity(version, commit, arch):
    if not re.fullmatch(r"[0-9a-f]{40}", commit) or arch not in ("amd64", "arm64"):
        raise ValueError("invalid source commit/architecture")
    if STABLE.fullmatch(version) or RC.fullmatch(version):
        tag = "v" + version
    # Historical preview downloads and local test snapshots remain readable, but
    # neither identity authorizes a new publication.
    elif version in ("preview-" + commit, "test-" + commit[:12]):
        tag = "preview-" + commit
    else:
        raise ValueError("invalid stable/RC/historical preview/test source version")
    name = f"mediagrap-source-{version}-{commit}-linux-{arch}.tar.gz"
    return tag, name, f"https://github.com/{REPOSITORY}/releases/download/{tag}/{name}"


def verify_archive(data, build, sha):
    if (
        not re.fullmatch(r"[0-9a-f]{64}", sha)
        or hashlib.sha256(data).hexdigest() != sha
    ):
        raise ValueError("source archive SHA256 mismatch")
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        members = archive.getmembers()
        names = [m.name for m in members]
        if len(names) != len(set(names)):
            raise ValueError("duplicate archive member")
        for member in members:
            path = PurePosixPath(member.name)
            if not member.isfile() or path.is_absolute() or ".." in path.parts:
                raise ValueError("unsafe archive member")

        def read(name):
            stream = archive.extractfile(name)
            if stream is None:
                raise ValueError("missing source material: " + name)
            return stream.read()

        manifest = json.loads(read("MANIFEST.json"))
        if any(
            manifest[key] != build[key] for key in ("version", "commit", "architecture")
        ):
            raise ValueError("archive/build identity mismatch")
        if set(names) != set(manifest["files"]) | {"MANIFEST.json"}:
            raise ValueError("archive inventory mismatch")
        for name, entry in manifest["files"].items():
            content = read(name)
            if (
                len(content) != entry["bytes"]
                or hashlib.sha256(content).hexdigest() != entry["sha256"]
            ):
                raise ValueError("source file checksum mismatch: " + name)
        source = json.loads(read("release-input.json"))
        if any(
            source[key] != manifest[key] for key in ("version", "commit", "testOnly")
        ):
            raise ValueError("source input identity mismatch")
        for name, sha256 in source["files"].items():
            if hashlib.sha256(read("mediagrap/" + name)).hexdigest() != sha256:
                raise ValueError("project source checksum mismatch: " + name)
        for name in (
            "mediagrap/LICENSE",
            "third-party/runtime/ffmpeg-7.1.1.tar.xz",
            "third-party/runtime/musl-1.2.5.tar.gz",
            "third-party/runtime/ca-certificates-20260611.tar.bz2",
            "third-party/runtime/ffmpeg/link.map",
            "third-party/runtime/ffmpeg/license.txt",
            "third-party/runtime/notices/GCC-COPYING.RUNTIME",
            "third-party/runtime/notices/GCC-COPYING3",
            "third-party/runtime/notices/MPL-2.0.txt",
            "third-party/runtime/notices/musl-COPYRIGHT",
            "third-party/go/Go-LICENSE",
            "third-party/go/modules.json",
            "third-party/npm/packages.json",
        ):
            if not read(name):
                raise ValueError("empty legal/source material: " + name)
        prefix = "third-party/frontend-upstream/"
        inventory = json.loads(read(prefix + "inventory.json"))
        if inventory != json.loads(read("mediagrap/docs/legal/frontend-sources.json")):
            raise ValueError("frontend inventory mismatch")
        for source in inventory["sources"]:
            verify_source(read(prefix + source["archive"]), source)
        for notice in inventory["bundledNotices"]:
            if (
                hashlib.sha256(read(prefix + notice["notice"])).hexdigest()
                != notice["sha256"]
            ):
                raise ValueError("bundled notice mismatch")
            if notice["sourceMarker"].encode() not in read(
                "third-party/npm/" + notice["bundledIn"] + "/" + notice["bundle"]
            ):
                raise ValueError("bundled source marker missing")
        return manifest, inventory


class HTTPSRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        if urlsplit(newurl).scheme != "https":
            raise ValueError("insecure source redirect")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def download_public(url):
    # Deliberately no GitHub token, cookies or authenticated API client.
    parsed = urlsplit(url)
    if (
        parsed.scheme != "https"
        or parsed.netloc != "github.com"
        or not parsed.path.startswith(f"/{REPOSITORY}/releases/download/")
        or parsed.query
        or parsed.fragment
    ):
        raise ValueError("not a pinned public source URL")
    opener = urllib.request.build_opener(HTTPSRedirect())
    with opener.open(url, timeout=120) as response:
        return response.read()
