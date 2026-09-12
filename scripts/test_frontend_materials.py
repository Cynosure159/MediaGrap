import hashlib
import importlib.util
import io
import json
import tarfile
import tempfile
import unittest
from pathlib import Path
from unittest import mock

spec = importlib.util.spec_from_file_location(
    "frontend_materials", Path(__file__).with_name("frontend_materials.py")
)
assert spec and spec.loader
materials = importlib.util.module_from_spec(spec)
spec.loader.exec_module(materials)


def source_fixture(version="4.6.4", include_build=True):
    data = io.BytesIO()
    files = {
        "fixture/packages/router/package.json": json.dumps(
            {"name": "vue-router", "version": version}
        ).encode(),
        "fixture/packages/router/src/index.ts": b"export const fixture = true",
    }
    if include_build:
        files["fixture/pnpm-lock.yaml"] = b"owned synthetic lock fixture"
    with tarfile.open(fileobj=data, mode="w:gz") as archive:
        for name, content in files.items():
            member = tarfile.TarInfo(name)
            member.size = len(content)
            archive.addfile(member, io.BytesIO(content))
    content = data.getvalue()
    source = {
        "archive": "fixture.tar.gz",
        "url": "https://codeload.github.com/vuejs/router/tar.gz/refs/tags/v4.6.4",
        "sha256": hashlib.sha256(content).hexdigest(),
        "root": "fixture",
        "version": "4.6.4",
        "packages": {"vue-router": "packages/router"},
        "requiredFiles": ["pnpm-lock.yaml", "packages/router/src/index.ts"],
    }
    return content, source


class FrontendMaterialsTests(unittest.TestCase):
    def test_preferred_source_requires_checksum_version_and_build_files(self):
        data, source = source_fixture()
        materials.verify_source(data, source)
        with self.assertRaisesRegex(ValueError, "checksum"):
            materials.verify_source(data + b"changed", source)
        data, source = source_fixture(version="4.6.5")
        with self.assertRaisesRegex(ValueError, "version mismatch"):
            materials.verify_source(data, source)
        data, source = source_fixture(include_build=False)
        with self.assertRaises(KeyError):
            materials.verify_source(data, source)

    def test_packaging_preserves_nested_bundle_and_supplies_verified_notice(self):
        self.package_fixture()

    def test_packaging_rejects_changed_nested_notice(self):
        with self.assertRaisesRegex(ValueError, "notice changed"):
            self.package_fixture(corrupt_notice=True)

    def test_packaging_rejects_changed_nested_dependency(self):
        with self.assertRaisesRegex(ValueError, "dependency version changed"):
            self.package_fixture(nested_version="6.6.5")

    def package_fixture(self, corrupt_notice=False, nested_version="6.6.4"):
        data, source = source_fixture()
        notice = b"owned synthetic upstream notice fixture"
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp) / "checkout"
            legal = root / "docs/legal"
            legal.mkdir(parents=True)
            (root / "web").mkdir()
            (root / "web/package-lock.json").write_text(
                json.dumps(
                    {
                        "packages": {
                            "node_modules/@vue/devtools-api": {
                                "version": nested_version
                            }
                        }
                    }
                )
            )
            inventory = {
                "sources": [source],
                "bundledNotices": [
                    {
                        "package": "@vue/devtools-api",
                        "version": "6.6.4",
                        "bundledIn": "vue-router",
                        "bundledInVersion": "4.6.4",
                        "bundle": "dist/vue-router.global.js",
                        "sourceMarker": "@vue/devtools-api/lib/esm/",
                        "notice": "nested-LICENSE",
                        "sha256": hashlib.sha256(notice).hexdigest(),
                    }
                ],
            }
            (legal / "frontend-sources.json").write_text(json.dumps(inventory))
            (legal / "nested-LICENSE").write_bytes(
                b"changed" if corrupt_notice else notice
            )
            package = Path(temp) / "package"
            bundle = package / "third-party/npm/vue-router/dist/vue-router.global.js"
            bundle.parent.mkdir(parents=True)
            bundle_data = (
                b"// @vue/devtools-api/lib/esm/index.js\nowned synthetic bundle"
            )
            bundle.write_bytes(bundle_data)
            response = mock.Mock(status=200)
            response.read.return_value = data
            with mock.patch.object(materials.http.client, "HTTPSConnection") as client:
                client.return_value.getresponse.return_value = response
                materials.package_frontend_materials(
                    root, package, [{"name": "vue-router", "version": "4.6.4"}]
                )
            target = package / "third-party/frontend-upstream"
            self.assertEqual((target / "fixture.tar.gz").read_bytes(), data)
            self.assertEqual((target / "nested-LICENSE").read_bytes(), notice)
            self.assertEqual(bundle.read_bytes(), bundle_data)
            self.assertEqual(
                json.loads((target / "inventory.json").read_text()), inventory
            )

    def test_reviewed_pins_match_lock_and_exact_notice(self):
        root = Path(__file__).resolve().parent.parent
        inventory = json.loads((root / "docs/legal/frontend-sources.json").read_text())
        lock = json.loads((root / "web/package-lock.json").read_text())["packages"]
        self.assertEqual(
            {s["version"] for s in inventory["sources"]}, {"3.5.42", "4.6.4"}
        )
        for source in inventory["sources"]:
            self.assertEqual(len(source["sha256"]), 64)
            for name in source["packages"]:
                self.assertEqual(
                    lock["node_modules/" + name]["version"], source["version"]
                )
        for notice in inventory["bundledNotices"]:
            content = (root / "docs/legal" / notice["notice"]).read_bytes()
            self.assertEqual(hashlib.sha256(content).hexdigest(), notice["sha256"])
            self.assertEqual(
                lock["node_modules/" + notice["package"]]["version"], notice["version"]
            )


if __name__ == "__main__":
    unittest.main()
