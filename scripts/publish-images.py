#!/usr/bin/env python3
"""Push saved tested images only after matching public source verification."""

import importlib.util
import json
import os
import subprocess
import sys
from pathlib import Path

spec = importlib.util.spec_from_file_location(
    "publication", Path(__file__).with_name("validate-publication.py")
)
assert spec and spec.loader
publication = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publication)


def docker(*args):
    return subprocess.check_output(["docker", *args]).decode().strip()


def remote_manifest(ref):
    result = subprocess.run(
        ["docker", "manifest", "inspect", ref], capture_output=True, text=True
    )
    if result.returncode:
        # Only explicit registry missing-manifest responses permit a first push.
        error = result.stderr.lower()
        if (
            ("manifest unknown" in error or "no such manifest" in error)
            and "unauthorized" not in error
            and "denied" not in error
        ):
            return None
        raise RuntimeError("registry manifest inspection failed")
    return json.loads(result.stdout)


def main():
    root = Path(sys.argv[1])
    version, branch = publication.current_identity()
    proofs = publication.validate_candidates(
        root, version, os.environ["EXPECTED_COMMIT"]
    )
    public = json.loads((root / "public-source-verification.json").read_text())
    for proof in proofs:
        if not any(
            item["url"] == proof["build"]["sourceURL"]
            and item["sha256"] == proof["archiveSHA256"]
            and item["bytes"] == proof["archiveBytes"]
            and item["anonymousGET"]
            for item in public
        ):
            raise ValueError("matching public source verification missing")
    image = os.environ["IMAGE"]
    refs = []
    for proof in proofs:
        arch = proof["architecture"]
        docker("load", "-i", str(root / ("distribution-" + arch) / "image.tar.gz"))
        local = "mediagrap:ci-" + arch
        if docker("image", "inspect", local, "--format", "{{.Id}}") != proof["imageID"]:
            raise ValueError("saved image differs from tested image")
        ref = image + ":" + version + "-" + arch
        existing = remote_manifest(ref)
        if existing is None:
            docker("tag", local, ref)
            docker("push", ref)
        elif existing.get("config", {}).get("digest") != proof["imageID"]:
            raise ValueError("refusing to overwrite immutable architecture tag")
        refs.append(ref)
    publication.current_identity()
    version_ref = image + ":" + version
    existing = remote_manifest(version_ref)
    if existing is None:
        docker("buildx", "imagetools", "create", "--tag", version_ref, *refs)
    else:
        manifests = existing.get("manifests", [])
        configs = set()
        for item in manifests:
            manifest = remote_manifest(image + "@" + item["digest"])
            if manifest is None:
                raise ValueError("existing index references missing manifest")
            configs.add(manifest["config"]["digest"])
        if len(manifests) != 2 or configs != {proof["imageID"] for proof in proofs}:
            raise ValueError("refusing to overwrite immutable multi-architecture tag")
    # The remote branch is checked immediately before moving the channel pointer.
    publication.current_identity()
    docker(
        "buildx",
        "imagetools",
        "create",
        "--tag",
        image + (":preview" if branch == "dev" else ":latest"),
        *refs,
    )


if __name__ == "__main__":
    main()
