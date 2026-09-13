#!/usr/bin/env python3
"""Export reviewed tracked source only; never send the checkout to a release builder."""

import argparse
import hashlib
import json
import re
import shutil
import subprocess
import tarfile
from pathlib import Path

SEMVER = re.compile(r"v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$")
FORBIDDEN = {
    ".git",
    ".local",
    ".vitest",
    ".pi",
    "data",
    "config",
    "cache",
    "runtime",
    "node_modules",
    "dist",
    "bin",
    "__pycache__",
}


def safe_path(name):
    if name == "internal/httpapi/ui/dist/.keep.html":
        return True
    path = Path(name)
    return (
        not path.is_absolute()
        and ".." not in path.parts
        and not any(ord(char) < 32 for char in name)
        and not any(part in FORBIDDEN or part.startswith(".env") for part in path.parts)
        and path.suffix.lower()
        not in {".db", ".sqlite", ".sqlite3", ".pem", ".key", ".tar", ".gz", ".zip"}
    )


def git(*args):
    return subprocess.check_output(["git", *args]).decode().strip()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", required=True)
    parser.add_argument("--tag")
    parser.add_argument("--test-snapshot", action="store_true")
    parser.add_argument("--preview", action="store_true")
    parser.add_argument("--test-extra", action="append", default=[])
    args = parser.parse_args()
    out = Path(args.output).resolve()
    root = Path(git("rev-parse", "--show-toplevel")).resolve()
    if out == root or root in out.parents:
        parser.error("output must be outside the checkout and must not exist")
    if out.exists():
        parser.error("output already exists")
    commit = git("rev-parse", "HEAD")
    if sum((args.test_snapshot, args.preview, bool(args.tag))) != 1:
        parser.error("choose exactly one of --tag, --preview, --test-snapshot")
    if args.test_snapshot:
        version = "test-" + commit[:12]
    else:
        if not args.preview and (not args.tag or not SEMVER.fullmatch(args.tag)):
            parser.error("release requires a stable vX.Y.Z tag")
        if args.test_extra or git("status", "--porcelain"):
            parser.error("release requires a clean tracked tree, no extra files")
        if args.preview:
            if git("symbolic-ref", "--short", "HEAD") != "dev":
                parser.error("preview requires the dev branch")
            version = "preview-" + commit
        else:
            if git("rev-parse", args.tag + "^{commit}") != commit:
                parser.error("tag must identify HEAD")
            version = args.tag[1:]
    paths = git("ls-files", "-z").split("\0")
    tracked = set(paths)
    paths += args.test_extra
    files = {}
    out.mkdir()
    try:
        for name in sorted(set(paths)):
            if not name:
                continue
            source = root / name
            # Test snapshots reflect tracked deletions without requiring a commit.
            # Explicitly requested extras and required build inputs still fail closed.
            if (
                args.test_snapshot
                and name in tracked
                and name not in args.test_extra
                and not source.exists()
                and not source.is_symlink()
            ):
                continue
            if (
                not safe_path(name)
                or source.is_symlink()
                or not source.resolve().is_relative_to(root)
                or not source.is_file()
            ):
                raise ValueError("unsafe or missing source path: " + name)
            destination = out / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            if args.test_snapshot:
                shutil.copyfile(source, destination)
            else:
                destination.write_bytes(
                    subprocess.check_output(["git", "show", "HEAD:" + name])
                )
            files[name] = hashlib.sha256(destination.read_bytes()).hexdigest()
        for required in (
            "LICENSE",
            "Dockerfile",
            "scripts/package-source.py",
            "scripts/build-ffprobe.sh",
        ):
            if required not in files:
                raise ValueError("required tracked build material missing: " + required)
        manifest = {
            "version": version,
            "commit": commit,
            "testOnly": args.test_snapshot,
            "builtAt": git("show", "-s", "--format=%cI", "HEAD"),
            "files": files,
        }
        with tarfile.open(out / "release-tree.tar", "w") as archive:
            for name in sorted(files):
                archive.add(out / name, arcname=name, recursive=False)
        (out / "release-input.json").write_text(json.dumps(manifest, indent=2) + "\n")
    except Exception:
        shutil.rmtree(out)
        raise
    print(
        json.dumps(
            {
                "context": str(out),
                "version": version,
                "commit": commit,
                "testOnly": args.test_snapshot,
            }
        )
    )


if __name__ == "__main__":
    main()
