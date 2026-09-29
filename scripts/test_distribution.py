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


class DockerSourceTests(unittest.TestCase):
    def test_ca_source_uses_official_distfiles_and_unchanged_checksum(self):
        dockerfile = (Path(__file__).resolve().parents[1] / "Dockerfile").read_text()
        archive = "ca-certificates-20260611.tar.bz2"
        self.assertIn(
            f"curl -fsSLo {archive} "
            f"https://distfiles.alpinelinux.org/distfiles/v3.22/{archive}",
            dockerfile,
        )
        self.assertIn(
            "echo '32ca73f2e81e2b88dc614f12e1ee04a82b1ec5a8e29d9f359ddf8905a0afcbb0"
            f"  {archive}' | sha256sum -c -",
            dockerfile,
        )
        # The distfiles tarball stays pinned for the notice inventory, but apk
        # packages are intentionally unpinned so certificates refresh with the
        # Alpine repository instead of failing dependency resolution.
        self.assertNotIn("https://gitlab.alpinelinux.org/", dockerfile)


class ReleaseContextTests(unittest.TestCase):
    def test_version_tags(self):
        for tag in ["v1.2.3", "v0.0.7", "v2.0.0", "v2.0.0-rc.1", "v1.2.3-rc.12"]:
            self.assertIsNotNone(release.RELEASE_TAG.fullmatch(tag), tag)
        for tag in [
            "1.2.3",
            "v01.2.3",
            "v1.2",
            "v1.2.3-01",
            "v2.0.0-rc.0",
            "v2.0.0-rc.01",
            "v2.0.0-rc1",
            "v2.0.0-rc.1-" + "a" * 40,
            "v2.0.0-rc.١",
            "preview-" + "a" * 40,
            "v1.2.3-alpha-1",
            "v1.2.3+local",
            "v1.2.3;echo unsafe",
        ]:
            self.assertIsNone(release.RELEASE_TAG.fullmatch(tag), tag)

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
        for tag in ("v1.2.3", "v1.2.3-rc.1"):
            with self.subTest(tag=tag):
                self.check_clean_export(tag)

    def check_clean_export(self, tag):
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
                ("rev-parse", "refs/tags/" + tag + "^{commit}"): commit,
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
                    ["release-context.py", "--tag", tag, "--output", str(output)],
                ),
                contextlib.redirect_stdout(io.StringIO()),
            ):
                release.main()
            self.assertEqual((output / "LICENSE").read_bytes(), b"committed bytes")
            self.assertFalse((output / "untracked.txt").exists())
            manifest = json.loads((output / "release-input.json").read_text())
            self.assertFalse(manifest["testOnly"])
            self.assertEqual(manifest["version"], tag[1:])
            self.assertEqual(manifest["commit"], commit)
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
                        tag,
                        "--output",
                        str(output.parent / "dirty"),
                    ],
                ),
                contextlib.redirect_stderr(io.StringIO()),
                self.assertRaises(SystemExit),
            ):
                release.main()

    def test_legacy_preview_export_is_rejected(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "checkout"
            root.mkdir()
            output = Path(temporary) / "preview"
            for branch, dirty, extra in (
                ("main", "", []),
                ("dev", " M LICENSE", []),
                ("dev", "", ["--test-snapshot"]),
            ):
                results = {
                    ("rev-parse", "--show-toplevel"): str(root),
                    ("rev-parse", "HEAD"): "a" * 40,
                    ("status", "--porcelain"): dirty,
                    ("symbolic-ref", "--short", "HEAD"): branch,
                }
                with (
                    mock.patch.object(
                        release,
                        "git",
                        side_effect=lambda *args, results=results: results[args],
                    ),
                    mock.patch(
                        "sys.argv",
                        [
                            "release-context.py",
                            "--preview",
                            "--output",
                            str(output),
                            *extra,
                        ],
                    ),
                    contextlib.redirect_stderr(io.StringIO()),
                    self.assertRaises(SystemExit),
                ):
                    release.main()
                self.assertFalse(output.exists())

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


if __name__ == "__main__":
    unittest.main()
