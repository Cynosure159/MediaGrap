#!/usr/bin/env python3
"""Recreate a build context from a downloaded, locally reviewed source archive."""

import argparse
import hashlib
import json
import shutil
import tarfile
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("archive")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    output = Path(args.output).resolve()
    if output.exists():
        parser.error("output must not exist")
    output.mkdir(parents=True)
    try:
        with tarfile.open(args.archive, "r:gz") as archive:

            def read(name):
                member = archive.getmember(name)
                if not member.isfile() or member.size > 256 * 1024 * 1024:
                    raise ValueError("unsafe archive member: " + name)
                stream = archive.extractfile(member)
                if stream is None:
                    raise ValueError("missing archive member: " + name)
                return stream.read()

            manifest = json.loads(read("MANIFEST.json"))
            # Check every supplied file before writing any project source.
            for name, entry in manifest["files"].items():
                if Path(name).is_absolute() or ".." in Path(name).parts:
                    raise ValueError("unsafe source path")
                data = read(name)
                if (
                    len(data) != entry["bytes"]
                    or hashlib.sha256(data).hexdigest() != entry["sha256"]
                ):
                    raise ValueError("source checksum mismatch: " + name)
            source = json.loads(read("release-input.json"))
            for name, sha in source["files"].items():
                if Path(name).is_absolute() or ".." in Path(name).parts:
                    raise ValueError("unsafe project path")
                data = read("mediagrap/" + name)
                if hashlib.sha256(data).hexdigest() != sha:
                    raise ValueError("project checksum mismatch: " + name)
                destination = output / name
                destination.parent.mkdir(parents=True, exist_ok=True)
                destination.write_bytes(data)
            (output / "release-input.json").write_bytes(read("release-input.json"))
            with tarfile.open(output / "release-tree.tar", "w") as tree:
                for name in sorted(source["files"]):
                    tree.add(output / name, arcname=name, recursive=False)
        print(
            "Verified source context ready; build with the archived Dockerfile. Registry/toolchain access is required."
        )
    except Exception:
        shutil.rmtree(output)
        raise


if __name__ == "__main__":
    main()
