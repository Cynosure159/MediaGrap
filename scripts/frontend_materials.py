"""Exact preferred Vue/router source and notices for code nested in npm bundles."""

import hashlib
import http.client
import io
import json
import tarfile
from urllib.parse import urlsplit


def verify_source(data, source):
    if hashlib.sha256(data).hexdigest() != source["sha256"]:
        raise ValueError(
            "upstream frontend source checksum changed: " + source["archive"]
        )
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:

        def read(name):
            member = archive.getmember(source["root"] + "/" + name)
            if not member.isfile() or member.size == 0:
                raise ValueError("preferred source material missing: " + name)
            stream = archive.extractfile(member)
            if stream is None:
                raise ValueError("preferred source material missing: " + name)
            return stream.read()

        for name in source["requiredFiles"]:
            read(name)
        for name, path in source["packages"].items():
            package = json.loads(read(path + "/package.json"))
            if package["name"] != name or package["version"] != source["version"]:
                raise ValueError("preferred source package/version mismatch: " + name)


def package_frontend_materials(root, package, npm_inventory):
    legal = root / "docs/legal"
    inventory = json.loads((legal / "frontend-sources.json").read_text())
    versions = {item["name"]: item["version"] for item in npm_inventory}
    destination = package / "third-party/frontend-upstream"
    destination.mkdir(parents=True)
    for source in inventory["sources"]:
        for name in source["packages"]:
            if name in versions and versions[name] != source["version"]:
                raise ValueError("preferred source/npm version mismatch: " + name)
        url = urlsplit(source["url"])
        if (
            url.scheme != "https"
            or url.netloc != "codeload.github.com"
            or not url.path.startswith("/vuejs/")
            or url.query
            or url.fragment
        ):
            raise ValueError("unexpected frontend source origin")
        connection = http.client.HTTPSConnection("codeload.github.com", timeout=120)
        try:
            connection.request("GET", url.path)
            response = connection.getresponse()
            if response.status != 200:
                raise ValueError(
                    "upstream frontend source HTTP status: " + str(response.status)
                )
            data = response.read()
        finally:
            connection.close()
        verify_source(data, source)
        (destination / source["archive"]).write_bytes(data)
    for notice in inventory["bundledNotices"]:
        name = notice["bundledIn"]
        if versions.get(name) != notice["bundledInVersion"]:
            raise ValueError("nested bundle inventory needs review: " + name)
        bundle = package / "third-party/npm" / name / notice["bundle"]
        if notice["sourceMarker"] not in bundle.read_text():
            raise ValueError("nested bundle source marker changed: " + name)
        lock = json.loads((root / "web/package-lock.json").read_text())["packages"]
        if lock["node_modules/" + notice["package"]]["version"] != notice["version"]:
            raise ValueError("nested dependency version changed: " + notice["package"])
        data = (legal / notice["notice"]).read_bytes()
        if hashlib.sha256(data).hexdigest() != notice["sha256"]:
            raise ValueError(
                "nested bundle upstream notice changed: " + notice["package"]
            )
        (destination / notice["notice"]).write_bytes(data)
    (destination / "inventory.json").write_text(json.dumps(inventory, indent=2) + "\n")
