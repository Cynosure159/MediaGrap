#!/usr/bin/env python3
"""Fail closed on channel/ref, tested image and matching source before login."""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

from source_contract import STABLE, source_identity, verify_archive


def git(*args):
    return subprocess.check_output(["git", *args]).decode().strip()


def publication_identity(event, ref, commit, remote_tip):
    if event != "push" or not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("publication requires a push and full commit")
    if ref == "refs/heads/dev":
        version = "preview-" + commit
        branch = "dev"
    elif ref.startswith("refs/tags/v") and STABLE.fullmatch(
        ref.removeprefix("refs/tags/v")
    ):
        version = ref.removeprefix("refs/tags/v")
        branch = "main"
        if git("rev-parse", ref + "^{commit}") != commit:
            raise ValueError("tag/commit mismatch")
    else:
        raise ValueError("only dev previews and stable vX.Y.Z tags may publish")
    if remote_tip(branch) != commit:
        raise ValueError("stale or wrong publication branch tip: " + branch)
    return version, branch


def current_identity():
    def remote_tip(branch):
        line = git("ls-remote", "--exit-code", "origin", "refs/heads/" + branch)
        return line.split()[0]

    return publication_identity(
        os.environ["GITHUB_EVENT_NAME"],
        os.environ["GITHUB_REF"],
        os.environ["EXPECTED_COMMIT"],
        remote_tip,
    )


def validate_candidates(root, version, commit):
    proofs = []
    for arch in ("amd64", "arm64"):
        directory = root / ("distribution-" + arch)
        proof = json.loads((directory / "verification.json").read_text())
        _, _, url = source_identity(version, commit, arch)
        build = proof["build"]
        if (
            proof["testOnly"]
            or proof["architecture"] != arch
            or proof["user"] != "65532:65532"
            or build["version"] != version
            or build["commit"] != commit
            or build["sourceURL"] != url
            or build["sourceArchitecture"] != arch
            or build["sourceSHA256"] != proof["archiveSHA256"]
            or not all(
                proof[key]
                for key in (
                    "sourceGETAndHEAD",
                    "sourcePathRejections",
                    "legacySourcesBookmarks",
                    "preferredFrontendSourcesAndNestedNotices",
                )
            )
        ):
            raise ValueError("candidate verification/identity mismatch: " + arch)
        data = (directory / "source.tar.gz").read_bytes()
        manifest, _ = verify_archive(
            data, {**build, "architecture": arch}, proof["archiveSHA256"]
        )
        if manifest["testOnly"] or len(data) != proof["archiveBytes"]:
            raise ValueError("test or mismatched source candidate")
        if not (directory / "image.tar.gz").is_file():
            raise ValueError("tested image missing: " + arch)
        proofs.append(proof)
    return proofs


def main():
    image = os.environ["IMAGE"]
    if not re.fullmatch(
        r"[a-z0-9]+(?:[._-][a-z0-9]+)*/[a-z0-9]+(?:[._-][a-z0-9]+)*", image
    ):
        raise ValueError("invalid DockerHub namespace/image")
    version, _ = current_identity()
    if len(sys.argv) > 1 and sys.argv[1] == "--guard-only":
        print(version)
        return
    validate_candidates(Path(sys.argv[1]), version, os.environ["EXPECTED_COMMIT"])
    print("Both tested architectures match the eligible channel, commit and source")


if __name__ == "__main__":
    main()
