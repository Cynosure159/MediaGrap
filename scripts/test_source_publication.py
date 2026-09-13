import hashlib
import importlib.util
import io
import json
import os
import subprocess
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

from source_contract import (
    HTTPSRedirect,
    download_public,
    source_identity,
    verify_archive,
)


def load(name):
    spec = importlib.util.spec_from_file_location(
        name, Path(__file__).with_name(name + ".py")
    )
    assert spec and spec.loader
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


publication = load("validate-publication")
publisher = load("publish-source")
images = load("publish-images")
release = load("release-context")


def fixture_archive(
    version="1.2.3", commit="a" * 40, arch="amd64", test_only=False, omit=None
):
    build = {"version": version, "commit": commit, "architecture": arch}
    files: dict[str, bytes] = dict.fromkeys(
        (
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
        ),
        b"owned synthetic legal fixture",
    )
    inventory = json.dumps({"sources": [], "bundledNotices": []}).encode()
    files["third-party/frontend-upstream/inventory.json"] = inventory
    files["mediagrap/docs/legal/frontend-sources.json"] = inventory
    files["release-input.json"] = json.dumps(
        {
            **build,
            "testOnly": test_only,
            "files": {
                "LICENSE": hashlib.sha256(files["mediagrap/LICENSE"]).hexdigest()
            },
        }
    ).encode()
    if omit:
        del files[omit]
    manifest = {
        **build,
        "testOnly": test_only,
        "files": {
            name: {"bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
            for name, data in files.items()
        },
    }
    files["MANIFEST.json"] = json.dumps(manifest).encode()
    raw = io.BytesIO()
    with tarfile.open(fileobj=raw, mode="w:gz") as archive:
        for name, data in files.items():
            member = tarfile.TarInfo(name)
            member.size = len(data)
            archive.addfile(member, io.BytesIO(data))
    return raw.getvalue(), build


def candidates(root, version="1.2.3", commit="a" * 40, test_only=False):
    proofs = []
    for arch in ("amd64", "arm64"):
        data, _ = fixture_archive(version, commit, arch, test_only)
        sha = hashlib.sha256(data).hexdigest()
        _, _, url = source_identity(version, commit, arch)
        proof = {
            "testOnly": test_only,
            "architecture": arch,
            "user": "65532:65532",
            "imageID": "sha256:" + arch,
            "build": {
                "version": version,
                "commit": commit,
                "sourceURL": url,
                "sourceArchitecture": arch,
                "sourceSHA256": sha,
            },
            "archiveSHA256": sha,
            "archiveBytes": len(data),
            "sourceGETAndHEAD": True,
            "sourcePathRejections": True,
            "legacySourcesBookmarks": True,
            "preferredFrontendSourcesAndNestedNotices": True,
        }
        directory = root / ("distribution-" + arch)
        directory.mkdir()
        (directory / "verification.json").write_text(json.dumps(proof))
        (directory / "source.tar.gz").write_bytes(data)
        (directory / "image.tar.gz").write_bytes(b"owned synthetic saved image")
        proofs.append(proof)
    return proofs


class SourceContractTests(unittest.TestCase):
    def test_pinned_identity(self):
        commit = "a" * 40
        for arch in ("amd64", "arm64"):
            for version, tag in (
                ("1.2.3", "v1.2.3"),
                ("preview-" + commit, "preview-" + commit),
                ("test-" + commit[:12], "preview-" + commit),
            ):
                actual_tag, name, url = source_identity(version, commit, arch)
                self.assertEqual(actual_tag, tag)
                self.assertIn(version + "-" + commit + "-linux-" + arch, name)
                self.assertEqual(
                    url,
                    f"https://github.com/Cynosure159/MediaGrap/releases/download/{tag}/{name}",
                )
        for version in (
            "latest",
            "1.2.3-rc.3",
            "preview-" + "b" * 40,
            "01.2.3",
            "1.2.3?x=1",
        ):
            with self.assertRaises(ValueError):
                source_identity(version, commit, "amd64")

    def test_exact_bytes_identity_and_legal_inventory(self):
        data, build = fixture_archive()
        sha = hashlib.sha256(data).hexdigest()
        verify_archive(data, build, sha)
        for altered, identity, digest in (
            (data + b"x", build, sha),
            (data, {**build, "architecture": "arm64"}, sha),
            (data, {**build, "commit": "b" * 40}, sha),
        ):
            with self.assertRaises(ValueError):
                verify_archive(altered, identity, digest)
        data, build = fixture_archive(omit="third-party/runtime/ffmpeg-7.1.1.tar.xz")
        with self.assertRaises(KeyError):
            verify_archive(data, build, hashlib.sha256(data).hexdigest())

    def test_download_is_unauthenticated_and_https_only(self):
        _, _, url = source_identity("1.2.3", "a" * 40, "amd64")
        response = mock.MagicMock()
        response.__enter__.return_value.read.return_value = b"owned fixture"
        opener = mock.Mock()
        opener.open.return_value = response
        with (
            mock.patch(
                "source_contract.urllib.request.build_opener", return_value=opener
            ),
            mock.patch.dict(os.environ, {"GH_TOKEN": "synthetic-not-a-credential"}),
        ):
            self.assertEqual(download_public(url), b"owned fixture")
            opener.open.assert_called_once_with(url, timeout=120)
        for unsafe in (
            "file:///tmp/source",
            "http://github.com/source",
            url + "?token=x",
            url.replace("github.com", "github.com.evil"),
        ):
            with self.assertRaises(ValueError):
                download_public(unsafe)
        with self.assertRaises(ValueError):
            HTTPSRedirect().redirect_request(
                mock.MagicMock(),
                io.BytesIO(),
                302,
                "",
                mock.MagicMock(),
                "http://example.test/source",
            )


class PublicationPolicyTests(unittest.TestCase):
    def test_synthetic_git_branch_and_tag_policy(self):
        # All Git history is isolated, fabricated via fast-import; the checkout is never staged or committed.
        with tempfile.TemporaryDirectory() as temporary:
            repo = Path(temporary) / "repo"
            subprocess.run(["git", "init", "-q", str(repo)], check=True)
            history = (
                "commit refs/heads/main\nmark :1\ncommitter Fixture <fixture@example.invalid> 1 +0000\ndata 4\nmain\n"
                "commit refs/heads/dev\nmark :2\ncommitter Fixture <fixture@example.invalid> 2 +0000\ndata 3\ndev\nfrom :1\n"
                "reset refs/tags/v1.2.3\nfrom :1\nreset refs/tags/v1.2.4\nfrom :2\n"
            )
            subprocess.run(
                ["git", "-C", str(repo), "fast-import", "--quiet"],
                input=history.encode(),
                check=True,
            )

            def git(*args):
                return (
                    subprocess.check_output(["git", "-C", str(repo), *args])
                    .decode()
                    .strip()
                )

            tips = {
                branch: git("rev-parse", "refs/heads/" + branch)
                for branch in ("main", "dev")
            }
            with mock.patch.object(publication, "git", side_effect=git):
                self.assertEqual(
                    publication.publication_identity(
                        "push", "refs/tags/v1.2.3", tips["main"], tips.__getitem__
                    ),
                    ("1.2.3", "main"),
                )
                self.assertEqual(
                    publication.publication_identity(
                        "push", "refs/heads/dev", tips["dev"], tips.__getitem__
                    ),
                    ("preview-" + tips["dev"], "dev"),
                )
                for event, ref, commit in (
                    ("push", "refs/tags/v1.2.4", tips["dev"]),
                    ("push", "refs/heads/main", tips["main"]),
                    ("push", "refs/heads/dev", tips["main"]),
                    ("pull_request", "refs/heads/dev", tips["dev"]),
                    ("push", "refs/tags/v1.0.0-rc.3", tips["main"]),
                    ("push", "refs/tags/v1.2.3", tips["dev"]),
                    ("push", "refs/heads/feature", tips["dev"]),
                ):
                    with self.assertRaises(ValueError):
                        publication.publication_identity(
                            event, ref, commit, tips.__getitem__
                        )

    def test_candidates_missing_test_wrong_hash_and_identity_fail_closed(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            proofs = candidates(root)
            publication.validate_candidates(root, "1.2.3", "a" * 40)
            path = root / "distribution-amd64/verification.json"
            for changed in (
                {"testOnly": True},
                {"architecture": "arm64"},
                {"archiveSHA256": "0" * 64},
                {"sourceGETAndHEAD": False},
            ):
                path.write_text(json.dumps({**proofs[0], **changed}))
                with self.assertRaises(ValueError):
                    publication.validate_candidates(root, "1.2.3", "a" * 40)
            path.write_text(json.dumps(proofs[0]))
            (root / "distribution-arm64/source.tar.gz").unlink()
            with self.assertRaises(FileNotFoundError):
                publication.validate_candidates(root, "1.2.3", "a" * 40)

    def test_upload_never_overwrites_and_checks_existing_public_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            proof = candidates(root)[0]
            tag, name, _ = source_identity("1.2.3", "a" * 40, "amd64")
            path = root / "distribution-amd64/source.tar.gz"
            with (
                mock.patch.object(publisher, "gh") as gh,
                mock.patch.object(
                    publisher, "download_public", return_value=path.read_bytes()
                ),
            ):
                publisher.ensure_asset(
                    tag,
                    name,
                    path,
                    {"assets": []},
                    proof["archiveSHA256"],
                    proof["build"],
                )
                self.assertEqual(gh.call_count, 1)
                self.assertNotIn("--clobber", gh.call_args.args)
                gh.reset_mock()
                publisher.ensure_asset(
                    tag,
                    name,
                    path,
                    {"assets": [{"name": name}]},
                    proof["archiveSHA256"],
                    proof["build"],
                )
                gh.assert_not_called()
            with (
                mock.patch.object(publisher, "gh") as gh,
                mock.patch.object(
                    publisher, "download_public", return_value=b"wrong existing bytes"
                ),
                self.assertRaises(ValueError),
            ):
                publisher.ensure_asset(
                    tag,
                    name,
                    path,
                    {"assets": [{"name": name}]},
                    proof["archiveSHA256"],
                    proof["build"],
                )
            gh.assert_not_called()
            with (
                mock.patch.object(
                    publisher, "gh", side_effect=RuntimeError("duplicate upload race")
                ),
                mock.patch.object(publisher, "download_public") as download,
                self.assertRaises(RuntimeError),
            ):
                publisher.ensure_asset(
                    tag,
                    name,
                    path,
                    {"assets": []},
                    proof["archiveSHA256"],
                    proof["build"],
                )
            download.assert_not_called()

    def test_images_require_public_asset_proof_before_docker(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            candidates(root)
            with (
                mock.patch.object(
                    images.publication,
                    "current_identity",
                    return_value=("1.2.3", "main"),
                ),
                mock.patch.dict(os.environ, {"EXPECTED_COMMIT": "a" * 40}),
                mock.patch("sys.argv", ["publish-images.py", temporary]),
                mock.patch.object(images, "docker") as docker,
                self.assertRaises(FileNotFoundError),
            ):
                images.main()
            docker.assert_not_called()

    def test_release_creation_and_anonymous_verification_for_both_channels(self):
        for branch in ("main", "dev"):
            with tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                commit = "a" * 40
                version = "1.2.3" if branch == "main" else "preview-" + commit
                candidates(root, version, commit)
                existing = {"draft": False, "prerelease": branch == "dev", "assets": []}

                def gh(*args, commit=commit):
                    return json.dumps({"sha": commit}) if args[0] == "api" else ""

                def download(url, root=root):
                    arch = "amd64" if "linux-amd64" in url else "arm64"
                    return (
                        root / ("distribution-" + arch) / "source.tar.gz"
                    ).read_bytes()

                with (
                    mock.patch.object(
                        publisher.publication,
                        "current_identity",
                        return_value=(version, branch),
                    ),
                    mock.patch.object(
                        publisher, "release_for", side_effect=[None, existing]
                    ),
                    mock.patch.object(publisher, "gh", side_effect=gh) as api,
                    mock.patch.object(
                        publisher, "download_public", side_effect=download
                    ),
                    mock.patch.dict(
                        os.environ,
                        {
                            "GITHUB_REPOSITORY": "Cynosure159/MediaGrap",
                            "EXPECTED_COMMIT": commit,
                        },
                    ),
                    mock.patch("sys.argv", ["publish-source.py", temporary]),
                ):
                    publisher.main()
                calls = [call.args for call in api.call_args_list]
                create = calls[0]
                self.assertEqual(create[:2], ("release", "create"))
                self.assertIn("--latest=false", create)
                self.assertEqual("--prerelease" in create, branch == "dev")
                self.assertEqual("--verify-tag" in create, branch == "main")
                self.assertEqual(
                    len([call for call in calls if call[:2] == ("release", "upload")]),
                    2,
                )
                report = json.loads(
                    (root / "public-source-verification.json").read_text()
                )
                self.assertEqual(len(report), 2)
                self.assertTrue(all(item["anonymousGET"] for item in report))

    def test_images_channel_stale_ref_and_immutable_mismatch(self):
        for branch, fault in (
            ("main", "none"),
            ("dev", "none"),
            ("dev", "stale"),
            ("main", "immutable"),
        ):
            with tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary)
                commit = "a" * 40
                version = "1.2.3" if branch == "main" else "preview-" + commit
                proofs = candidates(root, version, commit)
                public = [
                    {
                        "url": proof["build"]["sourceURL"],
                        "sha256": proof["archiveSHA256"],
                        "bytes": proof["archiveBytes"],
                        "anonymousGET": True,
                    }
                    for proof in proofs
                ]
                (root / "public-source-verification.json").write_text(
                    json.dumps(public)
                )

                def docker(*args):
                    if args[:2] == ("image", "inspect"):
                        return "sha256:" + args[2].removeprefix("mediagrap:ci-")
                    return ""

                identities = [
                    (version, branch),
                    (version, branch),
                    ValueError("stale tip") if fault == "stale" else (version, branch),
                ]
                existing = (
                    {"config": {"digest": "sha256:other"}}
                    if fault == "immutable"
                    else None
                )
                with (
                    mock.patch.object(
                        images.publication, "current_identity", side_effect=identities
                    ),
                    mock.patch.object(images, "remote_manifest", return_value=existing),
                    mock.patch.object(images, "docker", side_effect=docker) as commands,
                    mock.patch.dict(
                        os.environ,
                        {"EXPECTED_COMMIT": commit, "IMAGE": "example/mediagrap"},
                    ),
                    mock.patch("sys.argv", ["publish-images.py", temporary]),
                ):
                    if fault == "none":
                        images.main()
                    else:
                        with self.assertRaises(ValueError):
                            images.main()
                calls = [call.args for call in commands.call_args_list]
                mutable = "example/mediagrap:" + (
                    "latest" if branch == "main" else "preview"
                )
                self.assertEqual(
                    any(mutable in call for call in calls), fault == "none"
                )
                if branch == "dev":
                    self.assertFalse(
                        any("example/mediagrap:latest" in call for call in calls)
                    )
                if fault == "immutable":
                    self.assertFalse(any(call[0] in ("push", "tag") for call in calls))

    def test_registry_errors_are_not_missing_images(self):
        for error, missing in (
            ("no such manifest: example/image:1", True),
            ("manifest unknown", True),
            ("unauthorized: denied", False),
            ("dial tcp: host not found", False),
        ):
            result = subprocess.CompletedProcess([], 1, "", error)
            with mock.patch.object(images.subprocess, "run", return_value=result):
                if missing:
                    self.assertIsNone(images.remote_manifest("example/image:1"))
                else:
                    with self.assertRaises(RuntimeError):
                        images.remote_manifest("example/image:1")

    def test_workflow_native_checks_and_publication_only_credentials(self):
        text = (
            Path(__file__).resolve().parents[1] / ".github/workflows/ci.yml"
        ).read_text()
        self.assertIn("name: Go and web checks", text)
        self.assertIn("name: Docker (${{ matrix.arch }})", text)
        self.assertIn("runner: ubuntu-24.04-arm", text)
        self.assertIn("runner: ubuntu-24.04", text)
        self.assertIn("contents: read", text)
        verification, publish = text.split("\n  publish:\n")
        for forbidden in (
            "contents: write",
            "secrets.",
            "github.token",
            "docker/login",
            "scripts/publish-source.py",
            "scripts/publish-images.py",
        ):
            self.assertNotIn(forbidden, verification)
        for forbidden in ("setup-qemu", "pull_request_target"):
            self.assertNotIn(forbidden, text)
        self.assertIn("contents: write", publish)
        self.assertIn("github.event_name == 'push'", publish)
        self.assertIn("needs: [verify, docker]", publish)
        gates = [
            "scripts/validate-publication.py candidates",
            "scripts/publish-source.py candidates",
            "scripts/validate-publication.py --guard-only",
            "docker/login-action",
            "scripts/publish-images.py candidates",
        ]
        self.assertEqual(
            sorted(publish.index(gate) for gate in gates),
            [publish.index(gate) for gate in gates],
        )


if __name__ == "__main__":
    unittest.main()
