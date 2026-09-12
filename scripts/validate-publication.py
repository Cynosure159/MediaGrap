#!/usr/bin/env python3
"""Validate tested artifact identity before any registry login/publication."""

import importlib.util
import json
import os
import re
import subprocess
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "release_context", Path(__file__).with_name("release-context.py")
)
assert spec and spec.loader
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


def main():
    tag = os.environ["REF_NAME"]
    commit = os.environ["EXPECTED_COMMIT"]
    image = os.environ["IMAGE"]
    if not release.SEMVER.fullmatch(tag):
        raise ValueError("invalid vSemVer tag")
    if not re.fullmatch(
        r"[a-z0-9]+(?:[._-][a-z0-9]+)*/[a-z0-9]+(?:[._-][a-z0-9]+)*", image
    ):
        raise ValueError("invalid DockerHub namespace/image")
    actual = (
        subprocess.check_output(["git", "rev-parse", tag + "^{commit}"])
        .decode()
        .strip()
    )
    if actual != commit:
        raise ValueError("tag/commit mismatch")
    for arch in ["amd64", "arm64"]:
        directory = Path(sys.argv[1]) / ("distribution-" + arch)
        proof = json.loads((directory / "verification.json").read_text())
        if (
            proof["testOnly"]
            or proof["architecture"] != arch
            or proof["user"] != "65532:65532"
            or proof["build"]["version"] != tag[1:]
            or proof["build"]["commit"] != commit
        ):
            raise ValueError("candidate verification/identity mismatch: " + arch)
        if not (directory / "image.tar.gz").is_file():
            raise ValueError("tested image missing: " + arch)
    print("Both tested architectures match the release tag and commit")


if __name__ == "__main__":
    main()
