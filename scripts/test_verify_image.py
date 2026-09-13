import hashlib
import importlib.util
import io
import tarfile
import unittest
from pathlib import Path
from typing import Literal

spec = importlib.util.spec_from_file_location(
    "verify_image", Path(__file__).with_name("verify-image.py")
)
assert spec and spec.loader
verification = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verification)


def archive_bytes(files, mode: Literal["w", "w:gz"] = "w"):
    raw = io.BytesIO()
    with tarfile.open(fileobj=raw, mode=mode) as archive:
        for name, data in files.items():
            member = tarfile.TarInfo(name)
            member.size = len(data)
            archive.addfile(member, io.BytesIO(data))
    return raw.getvalue()


class VerifyImageFilesystemTests(unittest.TestCase):
    def test_immutable_image_components_fail_on_mismatch(self):
        files = {
            "mediagrap": b"x",
            "usr/bin/ffprobe": b"synthetic ffprobe",
            "etc/ssl/certs/ca-certificates.crt": b"synthetic public CA",
            "usr/share/mediagrap/LICENSE": b"synthetic notice",
        }
        checksum = "".join(
            hashlib.sha256(files[name]).hexdigest() + "  /" + name + "\n"
            for name in ("usr/bin/ffprobe", "etc/ssl/certs/ca-certificates.crt")
        )
        source = archive_bytes(
            {"third-party/runtime/runtime-sha256.txt": checksum.encode()}, "w:gz"
        )
        manifest = {
            "files": {
                "mediagrap/LICENSE": {
                    "sha256": hashlib.sha256(
                        files["usr/share/mediagrap/LICENSE"]
                    ).hexdigest()
                }
            }
        }
        exported = archive_bytes(files)
        proof = verification.verify_image_filesystem(exported, source, manifest)
        self.assertEqual(
            proof["imageCASHA256"],
            hashlib.sha256(files["etc/ssl/certs/ca-certificates.crt"]).hexdigest(),
        )
        for name in (
            "etc/ssl/certs/ca-certificates.crt",
            "usr/bin/ffprobe",
            "usr/share/mediagrap/LICENSE",
        ):
            with self.assertRaises(AssertionError):
                verification.verify_image_filesystem(
                    archive_bytes({**files, name: b"changed image bytes"}),
                    source,
                    manifest,
                )
        with self.assertRaises(AssertionError):
            verification.verify_image_filesystem(
                archive_bytes({**files, "source.tar.gz": b"retained source"}),
                source,
                manifest,
            )
        # A host-mutated started filesystem is evidence, not the distributed image.
        started = archive_bytes(
            {**files, "etc/ssl/certs/ca-certificates.crt": b"synthetic host CA"}
        )
        with tarfile.open(fileobj=io.BytesIO(started)) as filesystem:
            started_ca = verification.file_digest(
                filesystem, "etc/ssl/certs/ca-certificates.crt"
            )
        self.assertNotEqual(started_ca, proof["imageCASHA256"])
        self.assertEqual(
            verification.verify_image_filesystem(exported, source, manifest), proof
        )


if __name__ == "__main__":
    unittest.main()
