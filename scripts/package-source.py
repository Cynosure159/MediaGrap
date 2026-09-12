#!/usr/bin/env python3
"""Bake the verified build context plus exact dependency/build materials into /source."""

import gzip
import hashlib
import io
import json
import os
import shutil
import subprocess
import tarfile
import tempfile
import zipfile
from pathlib import Path

from frontend_materials import (
    package_frontend_materials,  # pyright: ignore[reportMissingImports]
)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def required(path):
    if not path.is_file() or path.stat().st_size == 0:
        raise ValueError("required distribution material missing: " + str(path))
    return path


def copy_tree(source, target):
    for path in sorted(source.rglob("*")):
        if "node_modules" in path.relative_to(source).parts:
            continue
        if path.is_symlink():
            raise ValueError("symlink in source materials: " + str(path))
        if path.is_file():
            dest = target / path.relative_to(source)
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path, dest)


def main():
    root = Path.cwd()
    manifest = json.loads(required(root / "release-input.json").read_text())
    for key, env in [("version", "VERSION"), ("commit", "COMMIT")]:
        if manifest[key] != os.environ[env]:
            raise ValueError("build/source " + key + " mismatch")
    materials = Path("/materials")
    notice_hashes = {
        "GCC-COPYING.RUNTIME": "9d6b43ce4d8de0c878bf16b54d8e7a10d9bd42b75178153e3af6a815bdc90f74",
        "GCC-COPYING3": "8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903",
        "MPL-2.0.txt": "fab3dd6bdab226f1c08630b1dd917e11fcb4ec5e1e020e2c16f83a0a13863e85",
    }
    for name, sha in notice_hashes.items():
        if digest(required(materials / "notices" / name)) != sha:
            raise ValueError("upstream legal text changed: " + name)
    required(materials / "notices/musl-COPYRIGHT")
    for name in [
        "ffmpeg-7.1.1.tar.xz",
        "musl-1.2.5.tar.gz",
        "ca-certificates-20260611.tar.bz2",
        "ffmpeg/config.h",
        "ffmpeg/config.mak",
        "ffmpeg/config.log",
        "ffmpeg/link.map",
        "ffmpeg/license.txt",
        "ffmpeg/buildconf.txt",
        "apk-packages.txt",
        "runtime-sha256.txt",
    ]:
        required(materials / name)
    if "LGPL version 2.1 or later" not in (materials / "ffmpeg/config.h").read_text():
        raise ValueError("unexpected ffprobe license")
    with tempfile.TemporaryDirectory() as temp:
        package = Path(temp)
        with tarfile.open(required(root / "release-tree.tar")) as tree:
            for name, sha in manifest["files"].items():
                member = tree.getmember(name)
                if (
                    not member.isfile()
                    or Path(name).is_absolute()
                    or ".." in Path(name).parts
                ):
                    raise ValueError("unsafe release tree entry")
                stream = tree.extractfile(member)
                if stream is None:
                    raise ValueError("release source is not a file")
                data = stream.read()
                if hashlib.sha256(data).hexdigest() != sha:
                    raise ValueError("release tree changed: " + name)
                source = root / name
                if source.is_file() and digest(source) != sha:
                    raise ValueError("build context changed: " + name)
                dest = package / "mediagrap" / name
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.write_bytes(data)
        shutil.copyfile(root / "release-input.json", package / "release-input.json")
        copy_tree(materials, package / "third-party" / "runtime")
        go = package / "third-party" / "go"
        go.mkdir(parents=True)
        # Whole module zip sources preserve nested notices (including modernc and MCP).
        output = subprocess.check_output(["go", "mod", "download", "-json"]).decode()
        decoder = json.JSONDecoder()
        modules = []
        while output.strip():
            module, end = decoder.raw_decode(output.lstrip())
            output = output.lstrip()[end:]
            if module.get("Error") or module.get("Replace"):
                raise ValueError("unresolved or replaced Go dependency")
            archive = required(Path(module["Zip"]))
            with zipfile.ZipFile(archive) as module_source:
                if not any(
                    Path(name).name.upper().startswith(("LICENSE", "COPYING"))
                    for name in module_source.namelist()
                ):
                    raise ValueError(
                        "Go module upstream license missing: " + module["Path"]
                    )
            name = module["Path"].replace("/", "_") + "@" + module["Version"] + ".zip"
            shutil.copyfile(archive, go / name)
            modules.append(
                {
                    "module": module["Path"],
                    "version": module["Version"],
                    "sha256": digest(archive),
                }
            )
        if not modules:
            raise ValueError("Go module sources missing")
        (go / "modules.json").write_text(json.dumps(modules, indent=2))
        goroot = Path(subprocess.check_output(["go", "env", "GOROOT"]).decode().strip())
        shutil.copyfile(required(goroot / "LICENSE"), go / "Go-LICENSE")
        (go / "toolchain.txt").write_bytes(subprocess.check_output(["go", "version"]))
        lock = json.loads((root / "web/package-lock.json").read_text())["packages"]
        inventory = json.loads(
            required(root / "web/dist/source-inventory.json").read_text()
        )
        # Workbox's generateSW build is separate from Vite: preserve every exact
        # Workbox package, not every npm build/test dependency.
        paths = {item["path"] for item in inventory}
        paths.update(
            key
            for key in lock
            if key.startswith("node_modules/workbox-") and key.count("/") == 1
        )
        npm_inventory = []
        for name in sorted(paths):
            source = root / "web" / name
            pkg = json.loads(required(source / "package.json").read_text())
            if pkg["version"] != lock[name]["version"]:
                raise ValueError("npm lock/source mismatch: " + name)
            if not any(
                p.is_file() and p.name.upper().startswith(("LICENSE", "COPYING"))
                for p in source.iterdir()
            ):
                raise ValueError("upstream npm notice missing: " + name)
            copy_tree(
                source, package / "third-party/npm" / name.removeprefix("node_modules/")
            )
            npm_inventory.append(
                {
                    "name": pkg["name"],
                    "version": pkg["version"],
                    "integrity": lock[name]["integrity"],
                }
            )
        (package / "third-party/npm/packages.json").write_text(
            json.dumps(npm_inventory, indent=2)
        )
        # Whole npm distributions can contain nested code absent from Vite output.
        # Supply its notice and preferred upstream source, not just compiled npm JS.
        package_frontend_materials(root, package, npm_inventory)
        checksums = {
            str(p.relative_to(package)): {
                "sha256": digest(p),
                "bytes": p.stat().st_size,
            }
            for p in sorted(package.rglob("*"))
            if p.is_file()
        }
        (package / "MANIFEST.json").write_text(
            json.dumps(
                {
                    "version": manifest["version"],
                    "commit": manifest["commit"],
                    "testOnly": manifest["testOnly"],
                    "files": checksums,
                },
                indent=2,
            )
            + "\n"
        )
        target = root / "internal/httpapi/distribution"
        target.mkdir(parents=True, exist_ok=True)
        with (
            (target / "source.tar.gz").open("wb") as raw,
            gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=0) as compressed,
            tarfile.open(fileobj=compressed, mode="w") as tar,
        ):
            for path in sorted(package.rglob("*")):
                if not path.is_file():
                    continue
                data = path.read_bytes()
                info = tarfile.TarInfo(str(path.relative_to(package)))
                info.size, info.mode, info.mtime = len(data), 0o644, 0
                tar.addfile(info, io.BytesIO(data))
        archive = target / "source.tar.gz"
        (target / "manifest.json").write_text(
            json.dumps(
                {
                    "version": manifest["version"],
                    "commit": manifest["commit"],
                    "testOnly": manifest["testOnly"],
                    "sha256": digest(archive),
                    "bytes": archive.stat().st_size,
                },
                indent=2,
            )
            + "\n"
        )
        print((target / "manifest.json").read_text())


if __name__ == "__main__":
    main()
