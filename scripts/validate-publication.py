#!/usr/bin/env python3
"""Fail closed on channel/ref, tested image and matching source before login."""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

from source_contract import RC, RELEASE_TAG, source_identity, verify_archive


def git(*args):
    return subprocess.check_output(["git", *args]).decode().strip()


def publication_identity(event, ref, commit, remote_tip):
    if event != "push" or not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("publication requires a push and full commit")
    if not ref.startswith("refs/tags/") or not RELEASE_TAG.fullmatch(
        ref.removeprefix("refs/tags/")
    ):
        raise ValueError("only vX.Y.Z-rc.N and stable vX.Y.Z tag pushes may publish")
    version = ref.removeprefix("refs/tags/v")
    branch = "dev" if RC.fullmatch(version) else "main"
    if git("rev-parse", ref + "^{commit}") != commit:
        raise ValueError("tag/commit mismatch")
    if remote_tip(branch) != commit:
        raise ValueError("stale or wrong publication branch tip: " + branch)
    return version, branch


def current_identity():
    def remote_tip(branch):
        line = git("ls-remote", "--exit-code", "origin", "refs/heads/" + branch)
        return line.split()[0]

    ref = os.environ["GITHUB_REF"]
    commit = os.environ["EXPECTED_COMMIT"]
    identity = publication_identity(
        os.environ["GITHUB_EVENT_NAME"], ref, commit, remote_tip
    )
    # Check the live tag too, peeling annotated tags. A moved/deleted remote tag
    # must not publish merely because the checkout still has the original ref.
    refs = {
        name: sha
        for sha, name in (
            line.split()
            for line in git(
                "ls-remote", "--exit-code", "origin", ref, ref + "^{}"
            ).splitlines()
        )
    }
    if refs.get(ref + "^{}", refs.get(ref)) != commit:
        raise ValueError("remote tag/commit mismatch")
    return identity


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
