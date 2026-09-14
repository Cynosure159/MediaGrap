#!/usr/bin/env python3
"""Publication-only: retain immutable Release assets, then verify anonymous GETs.

Never run from verification/PR jobs. Requires the gated job's GH_TOKEN only for
GitHub API writes; download_public deliberately does not use that credential.
"""

import importlib.util
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
from pathlib import Path

from source_contract import (
    RELEASE_TAG,
    REPOSITORY,
    download_public,
    source_identity,
    verify_archive,
)

spec = importlib.util.spec_from_file_location(
    "publication", Path(__file__).with_name("validate-publication.py")
)
assert spec and spec.loader
publication = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publication)


def gh(*args):
    return subprocess.check_output(["gh", *args], stderr=subprocess.PIPE).decode()


RELEASE_VISIBILITY_ATTEMPTS = 6


def release_for(tag):
    # source_identity produces a path-safe tag; keep this boundary explicit before
    # placing it in the exact-tag API path. Only an actual HTTP 404 means missing.
    if not RELEASE_TAG.fullmatch(tag):
        raise ValueError("invalid release tag")
    try:
        return json.loads(gh("api", f"repos/{REPOSITORY}/releases/tags/{tag}"))
    except subprocess.CalledProcessError as error:
        raw_detail = error.stderr or ""
        detail = (
            raw_detail.decode(errors="replace")
            if isinstance(raw_detail, bytes)
            else str(raw_detail)
        )
        if "HTTP 404" in detail:
            return None
        raise


def release_after_create(tag):
    for attempt in range(RELEASE_VISIBILITY_ATTEMPTS):
        existing = release_for(tag)
        if existing is not None:
            return existing
        if attempt + 1 < RELEASE_VISIBILITY_ATTEMPTS:
            time.sleep(5 * (attempt + 1))
    return None


def ensure_asset(tag, name, path, existing, expected_sha, build):
    if name not in {asset["name"] for asset in existing["assets"]}:
        # gh upload fails on a duplicate name. Never delete/overwrite to resume.
        gh("release", "upload", tag, str(path), "--repo", REPOSITORY)
    _, _, url = source_identity(
        build["version"], build["commit"], build["sourceArchitecture"]
    )
    for attempt in range(6):
        try:
            data = download_public(url)
            break
        except OSError:
            if attempt == 5:
                raise
            time.sleep(5 * (attempt + 1))
    else:
        raise RuntimeError("public source download attempts exhausted")
    manifest, _ = verify_archive(
        data, {**build, "architecture": build["sourceArchitecture"]}, expected_sha
    )
    if manifest["testOnly"]:
        raise ValueError("test-only public asset")
    return {
        "url": url,
        "sha256": expected_sha,
        "bytes": len(data),
        "anonymousGET": True,
    }


def main():
    if os.environ["GITHUB_REPOSITORY"] != REPOSITORY:
        raise ValueError("publication is pinned to the actual source repository")
    version, branch = publication.current_identity()
    commit = os.environ["EXPECTED_COMMIT"]
    root = Path(sys.argv[1])
    proofs = publication.validate_candidates(root, version, commit)
    tag, _, _ = source_identity(version, commit, "amd64")
    existing = release_for(tag)
    if existing is None:
        args = [
            "release",
            "create",
            tag,
            "--repo",
            REPOSITORY,
            "--target",
            commit,
            "--title",
            tag,
            "--notes",
            "Matching source for the verified MediaGrap images. Retain these assets while distributing the binaries.",
            "--latest=false",
            "--verify-tag",
        ]
        if branch == "dev":
            args.append("--prerelease")
        gh(*args)
        existing = release_after_create(tag)
        if existing is None:
            raise ValueError("created release not visible after bounded retries")
    if existing["draft"] or existing["prerelease"] != (branch == "dev"):
        raise ValueError("release channel/draft mismatch")
    # Resolve annotated or lightweight tags through the GitHub commit API.
    tagged = json.loads(gh("api", f"repos/{REPOSITORY}/commits/{tag}"))["sha"]
    if tagged != commit:
        raise ValueError("existing release tag points at another commit")
    downloads = []
    with tempfile.TemporaryDirectory() as temporary:
        for proof in proofs:
            arch = proof["architecture"]
            _, name, _ = source_identity(version, commit, arch)
            path = Path(temporary) / name
            shutil.copyfile(root / ("distribution-" + arch) / "source.tar.gz", path)
            downloads.append(
                ensure_asset(
                    tag, name, path, existing, proof["archiveSHA256"], proof["build"]
                )
            )
    (root / "public-source-verification.json").write_text(
        json.dumps(downloads, indent=2) + "\n"
    )
    print(
        "Both matching source assets are public and hash/identity verified; image publication may proceed"
    )


if __name__ == "__main__":
    main()
