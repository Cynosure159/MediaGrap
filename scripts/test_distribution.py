import contextlib
import importlib.util
import io
import json
import tempfile
import unittest
from pathlib import Path
from unittest import mock

spec = importlib.util.spec_from_file_location(
    "release_context", Path(__file__).with_name("release-context.py")
)
assert spec and spec.loader
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleaseContextTests(unittest.TestCase):
    def test_version_tags(self):
        for tag in ["v1.2.3", "v0.0.7", "v2.0.0-rc.1", "v1.2.3-alpha-1"]:
            self.assertIsNotNone(release.SEMVER.fullmatch(tag), tag)
        for tag in [
            "1.2.3",
            "v01.2.3",
            "v1.2",
            "v1.2.3-01",
            "v1.2.3+local",
            "v1.2.3;echo unsafe",
        ]:
            self.assertIsNone(release.SEMVER.fullmatch(tag), tag)

    def test_source_exclusions(self):
        for name in [
            ".env",
            ".env.local",
            ".git/config",
            "config/session.key",
            "data/a.db",
            "cache/token",
            ".vitest/test.json",
            "../outside",
            "/outside",
            "web/node_modules/a.js",
            "dist/source.tar.gz",
            "backup.sqlite3",
            ".local/a",
            "runtime/config/a",
        ]:
            self.assertFalse(release.safe_path(name), name)
        for name in [
            "Dockerfile",
            "LICENSE",
            "scripts/build-ffprobe.sh",
            "web/package-lock.json",
            "internal/httpapi/ui/dist/.keep.html",
        ]:
            self.assertTrue(release.safe_path(name), name)

    def test_clean_export_uses_commit_blobs_not_working_bytes(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "checkout"
            root.mkdir()
            names = [
                "LICENSE",
                "Dockerfile",
                "scripts/package-source.py",
                "scripts/build-ffprobe.sh",
            ]
            for name in names:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("working bytes")
            (root / "untracked.txt").write_text("synthetic exclusion fixture")
            commit = "a" * 40
            results = {
                ("rev-parse", "--show-toplevel"): str(root),
                ("rev-parse", "HEAD"): commit,
                ("status", "--porcelain"): "",
                ("rev-parse", "v1.2.3^{commit}"): commit,
                ("ls-files", "-z"): "\0".join(names),
                ("show", "-s", "--format=%cI", "HEAD"): "2026-01-01T00:00:00Z",
            }
            output = Path(temporary) / "export"
            with (
                mock.patch.object(
                    release, "git", side_effect=lambda *args: results[args]
                ),
                mock.patch.object(
                    release.subprocess, "check_output", return_value=b"committed bytes"
                ),
                mock.patch(
                    "sys.argv",
                    ["release-context.py", "--tag", "v1.2.3", "--output", str(output)],
                ),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                release.main()
            self.assertEqual((output / "LICENSE").read_bytes(), b"committed bytes")
            self.assertFalse((output / "untracked.txt").exists())
            self.assertFalse(
                json.loads((output / "release-input.json").read_text())["testOnly"]
            )
            results[("status", "--porcelain")] = " M LICENSE"
            with (
                mock.patch.object(
                    release, "git", side_effect=lambda *args: results[args]
                ),
                mock.patch(
                    "sys.argv",
                    [
                        "release-context.py",
                        "--tag",
                        "v1.2.3",
                        "--output",
                        str(output.parent / "dirty"),
                    ],
                ),
                contextlib.redirect_stderr(io.StringIO()),
                self.assertRaises(SystemExit),
            ):
                release.main()

    def test_snapshot_handles_tracked_deletions_but_not_missing_required_inputs(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "checkout"
            root.mkdir()
            names = [
                "LICENSE",
                "Dockerfile",
                "scripts/package-source.py",
                "scripts/build-ffprobe.sh",
            ]
            for name in names:
                path = root / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("owned tracked fixture")
            results = {
                ("rev-parse", "--show-toplevel"): str(root),
                ("rev-parse", "HEAD"): "a" * 40,
                ("ls-files", "-z"): "\0".join([*names, "docs/retired.md"]),
                ("show", "-s", "--format=%cI", "HEAD"): "2026-01-01T00:00:00Z",
            }
            output = Path(temporary) / "snapshot"
            with (
                mock.patch.object(
                    release, "git", side_effect=lambda *args: results[args]
                ),
                mock.patch(
                    "sys.argv",
                    ["release-context.py", "--test-snapshot", "--output", str(output)],
                ),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                release.main()
            manifest = json.loads((output / "release-input.json").read_text())
            self.assertTrue(manifest["testOnly"])
            self.assertNotIn("docs/retired.md", manifest["files"])
            (root / "LICENSE").unlink()
            with (
                mock.patch.object(
                    release, "git", side_effect=lambda *args: results[args]
                ),
                mock.patch(
                    "sys.argv",
                    [
                        "release-context.py",
                        "--test-snapshot",
                        "--output",
                        str(output.parent / "missing"),
                    ],
                ),
                self.assertRaisesRegex(
                    ValueError, "required tracked build material missing"
                ),
            ):
                release.main()

    def test_publication_rejects_test_artifacts_before_login(self):
        publication_spec = importlib.util.spec_from_file_location(
            "publication", Path(__file__).with_name("validate-publication.py")
        )
        assert publication_spec and publication_spec.loader
        publication = importlib.util.module_from_spec(publication_spec)
        publication_spec.loader.exec_module(publication)
        with tempfile.TemporaryDirectory() as temporary:
            for arch in ["amd64", "arm64"]:
                directory = Path(temporary) / ("distribution-" + arch)
                directory.mkdir()
                proof = {
                    "testOnly": False,
                    "architecture": arch,
                    "user": "65532:65532",
                    "build": {"version": "1.2.3", "commit": "a" * 40},
                }
                (directory / "verification.json").write_text(json.dumps(proof))
                (directory / "image.tar.gz").write_bytes(
                    b"owned synthetic image fixture"
                )
            with (
                mock.patch.dict(
                    "os.environ",
                    {
                        "REF_NAME": "v1.2.3",
                        "EXPECTED_COMMIT": "a" * 40,
                        "IMAGE": "example/mediagrap",
                    },
                ),
                mock.patch.object(
                    publication.subprocess,
                    "check_output",
                    return_value=("a" * 40).encode(),
                ),
                mock.patch("sys.argv", ["validate-publication.py", temporary]),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                publication.main()
                proof["testOnly"] = True
                (directory / "verification.json").write_text(json.dumps(proof))
                with self.assertRaises(ValueError):
                    publication.main()


if __name__ == "__main__":
    unittest.main()
