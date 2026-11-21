#!/usr/bin/env python3
"""How a Trenova release is built: which images it builds, which it carries
forward, and the release manifest it publishes (docs/engineering/releases.md).

  release.py check-inputs
      fails when a Dockerfile copies a path its entry in
      deploy/release/images.yaml does not list, or a workflow the release runs
      builds an image the list does not name

  release.py plan --tag v0.13.1 [--rebuild all|a,b] --out plan.json
      decides, per image, build or reuse (carry the previous digest forward),
      and writes the plan; with GITHUB_OUTPUT set, also the job outputs

  release.py image <name> --plan plan.json
      the build parameters of one image, as job outputs

  release.py aliases --plan plan.json [--run]
      the `docker buildx imagetools create` commands that give every reused
      image this release's tags; --run runs them

  release.py manifest --plan plan.json --out release-manifest.json
      the release manifest, once every image is on the registry

  release.py pin-charts --manifest release-manifest.json [--previous DIR] [--refresh]
      writes release-manifest.json and image-digests.yaml into both charts and
      points the chart's PostgreSQL tag at the manifest's

  release.py validate release-manifest.json

Only the Python standard library and PyYAML (on every GitHub runner) are used.
"""

from __future__ import annotations

import argparse
import base64
import dataclasses
import json
import os
import re
import shlex
import subprocess
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Callable, Iterable, Protocol

import yaml

ROOT = Path(__file__).resolve().parents[2]
IMAGES_FILE = "deploy/release/images.yaml"
VALUES_FILE = "deploy/helm/trenova/values.yaml"
MIGRATIONS = "services/tms/schema/migrations/"
CHARTS = ("trenova", "trenova-operator")
MANIFEST_NAME = "release-manifest.json"
DIGESTS_NAME = "image-digests.yaml"
SCHEMA_VERSION = 1

VERSION = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
TAG = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
DIGEST = re.compile(r"sha256:[0-9a-f]{64}")
COMMIT = re.compile(r"[0-9a-f]{40}")
NAME = re.compile(r"[a-z][a-z0-9-]*")

BUILD, REUSE, PIN = "build", "reuse", "pin"


class ReleaseError(Exception):
    """A release that cannot go on, with the reason a person reads."""


def key(version: str) -> tuple[int, int, int]:
    return tuple(int(p) for p in version.split("."))  # type: ignore[return-value]


# The image list ---------------------------------------------------------------


@dataclasses.dataclass(frozen=True)
class Image:
    name: str
    repository: str
    dockerfile: str = ""
    context: str = ""
    target: str = ""
    platforms: tuple[str, ...] = ()
    build_args: tuple[tuple[str, str], ...] = ()
    image_args: tuple[tuple[str, str], ...] = ()
    depends_on: tuple[str, ...] = ()
    lagging: bool = False
    inputs: tuple[str, ...] = ()
    pinned: str = ""

    @property
    def built(self) -> bool:
        return not self.pinned


def load_images(text: str) -> dict[str, Image]:
    doc = yaml.safe_load(text) or {}
    raw = doc.get("images")
    if not isinstance(raw, dict) or not raw:
        raise ReleaseError(f"{IMAGES_FILE}: no images")
    images: dict[str, Image] = {}
    for name, entry in raw.items():
        if not isinstance(name, str) or not NAME.fullmatch(name):
            raise ReleaseError(f"{IMAGES_FILE}: {name!r} is not an image name")
        if not isinstance(entry, dict):
            raise ReleaseError(f"{IMAGES_FILE}: {name} is not a mapping")
        known = {
            "repository", "dockerfile", "context", "target", "platforms", "buildArgs",
            "imageArgs", "dependsOn", "lagging", "inputs", "pinned",
        }
        unknown = set(entry) - known
        if unknown:
            raise ReleaseError(f"{IMAGES_FILE}: {name} has unknown keys {sorted(unknown)}")
        repository = entry.get("repository")
        if not isinstance(repository, str) or not repository:
            raise ReleaseError(f"{IMAGES_FILE}: {name} has no repository")
        if entry.get("pinned"):
            extra = known - {"repository", "pinned"}
            if extra & set(entry):
                raise ReleaseError(f"{IMAGES_FILE}: {name} is pinned and cannot say how it is built")
            images[name] = Image(name=name, repository=repository, pinned=str(entry["pinned"]))
            continue
        for field in ("dockerfile", "context"):
            if not isinstance(entry.get(field), str) or not entry[field]:
                raise ReleaseError(f"{IMAGES_FILE}: {name} has no {field}")
        platforms = entry.get("platforms") or []
        inputs = entry.get("inputs") or []
        if not platforms or not all(isinstance(p, str) for p in platforms):
            raise ReleaseError(f"{IMAGES_FILE}: {name} has no platforms")
        if not inputs or not all(isinstance(p, str) and p for p in inputs):
            raise ReleaseError(f"{IMAGES_FILE}: {name} has no inputs")
        if not _covered(entry["dockerfile"], inputs):
            raise ReleaseError(f"{IMAGES_FILE}: {name} does not list its Dockerfile as an input")
        images[name] = Image(
            name=name,
            repository=repository,
            dockerfile=entry["dockerfile"],
            context=entry["context"],
            target=entry.get("target") or "",
            platforms=tuple(platforms),
            build_args=tuple((str(k), str(v)) for k, v in (entry.get("buildArgs") or {}).items()),
            image_args=tuple((str(k), str(v)) for k, v in (entry.get("imageArgs") or {}).items()),
            depends_on=tuple(entry.get("dependsOn") or ()),
            lagging=bool(entry.get("lagging")),
            inputs=tuple(inputs),
        )
    for image in images.values():
        for dep in set(image.depends_on) | {v for _, v in image.image_args}:
            target = images.get(dep)
            if target is None or not target.built:
                raise ReleaseError(f"{IMAGES_FILE}: {image.name} depends on {dep}, which is not built here")
            if target.depends_on:
                raise ReleaseError(f"{IMAGES_FILE}: {image.name} depends on {dep}, which depends on another image")
        for _, dep in image.image_args:
            if dep not in image.depends_on:
                raise ReleaseError(f"{IMAGES_FILE}: {image.name} takes {dep}'s reference but does not depend on it")
    return images


# Ports: git, the registry, the previous release's manifest -----------------------


class Git(Protocol):
    def commit(self, ref: str) -> str: ...
    def tags(self, commit: str) -> list[str]: ...
    def all_tags(self) -> list[str]: ...
    def changed(self, base: str, head: str, paths: Iterable[str]) -> list[str]: ...
    def show(self, ref: str, path: str) -> str: ...


class Registry(Protocol):
    def digest(self, repository: str, reference: str) -> str | None: ...
    def tags(self, repository: str) -> list[str]: ...


class Releases(Protocol):
    def manifest(self, tag: str) -> dict[str, Any] | None: ...


class LocalGit:
    def __init__(self, root: Path = ROOT):
        self.root = root

    def _run(self, *args: str) -> str:
        return subprocess.run(
            ["git", *args], cwd=self.root, check=True, capture_output=True, text=True
        ).stdout

    def commit(self, ref: str) -> str:
        return self._run("rev-parse", f"{ref}^{{commit}}").strip()

    def tags(self, commit: str) -> list[str]:
        return self._run("tag", "--merged", commit, "--list", "v*").split()

    def all_tags(self) -> list[str]:
        return self._run("tag", "--list", "v*").split()

    def changed(self, base: str, head: str, paths: Iterable[str]) -> list[str]:
        out = self._run("diff", "--name-only", base, head, "--", *paths)
        return [line for line in out.splitlines() if line]

    def show(self, ref: str, path: str) -> str:
        return self._run("show", f"{ref}:{path}")


class GHCR:
    """The registry API of ghcr.io, with the job's token (GHCR refuses the
    runners' anonymous token requests)."""

    ACCEPT = ", ".join([
        "application/vnd.oci.image.index.v1+json",
        "application/vnd.docker.distribution.manifest.list.v2+json",
        "application/vnd.oci.image.manifest.v1+json",
        "application/vnd.docker.distribution.manifest.v2+json",
    ])

    def __init__(self, user: str, token: str):
        self.basic = base64.b64encode(f"{user}:{token}".encode()).decode()
        self.tokens: dict[str, str] = {}

    @staticmethod
    def split(repository: str) -> tuple[str, str]:
        host, _, path = repository.partition("/")
        if host != "ghcr.io" or not path:
            raise ReleaseError(f"{repository} is not on ghcr.io")
        return host, path

    def _token(self, host: str, path: str) -> str:
        if path not in self.tokens:
            request = urllib.request.Request(
                f"https://{host}/token?scope=repository:{path}:pull",
                headers={"Authorization": f"Basic {self.basic}", "User-Agent": "trenova-release"},
            )
            with urllib.request.urlopen(request, timeout=30) as response:
                self.tokens[path] = json.load(response)["token"]
        return self.tokens[path]

    def _request(self, method: str, repository: str, url_path: str, accept: str = "") -> Any:
        host, path = self.split(repository)
        try:
            token = self._token(host, path)
        except urllib.error.HTTPError as e:
            if e.code in (403, 404):
                return None
            raise
        headers = {"Authorization": f"Bearer {token}", "User-Agent": "trenova-release"}
        if accept:
            headers["Accept"] = accept
        request = urllib.request.Request(
            f"https://{host}/v2/{path}/{url_path}", headers=headers, method=method
        )
        try:
            return urllib.request.urlopen(request, timeout=30)
        except urllib.error.HTTPError as e:
            # GHCR answers a missing repository with 403 or 404 and a missing
            # tag or digest with 404.
            if e.code in (403, 404):
                return None
            raise

    def digest(self, repository: str, reference: str) -> str | None:
        response = self._request("HEAD", repository, f"manifests/{reference}", self.ACCEPT)
        if response is None:
            return None
        with response:
            digest = response.headers.get("Docker-Content-Digest", "")
        if not DIGEST.fullmatch(digest):
            raise ReleaseError(f"{repository}:{reference}: the registry gave no digest")
        return digest

    def tags(self, repository: str) -> list[str]:
        response = self._request("GET", repository, "tags/list?n=10000")
        if response is None:
            return []
        with response:
            return json.load(response).get("tags") or []


class GitHubReleases:
    """The previous release's manifest, read with the GitHub CLI."""

    def manifest(self, tag: str) -> dict[str, Any] | None:
        view = subprocess.run(
            ["gh", "release", "view", tag, "--json", "assets", "--jq", ".assets[].name"],
            cwd=ROOT, capture_output=True, text=True,
        )
        if view.returncode != 0:
            if "release not found" in view.stderr:
                return None
            raise ReleaseError(f"cannot read the GitHub release {tag}: {view.stderr.strip()}")
        assets = view.stdout.split()
        if MANIFEST_NAME not in assets:
            return None
        body = subprocess.run(
            ["gh", "release", "download", tag, "--pattern", MANIFEST_NAME, "--output", "-"],
            cwd=ROOT, check=True, capture_output=True, text=True,
        ).stdout
        manifest = json.loads(body)
        validate_manifest(manifest)
        return manifest


# The plan ------------------------------------------------------------------------


def parse_rebuild(text: str, images: dict[str, Image]) -> tuple[set[str], bool]:
    """The images a re-run names; `all` names every built image and refreshes
    the third-party digests the charts pin."""
    names = {n.strip() for n in (text or "").split(",") if n.strip()}
    if "all" in names:
        if names != {"all"}:
            raise ReleaseError("rebuild: `all` stands alone")
        return {n for n, i in images.items() if i.built}, True
    for name in sorted(names):
        image = images.get(name)
        if image is None:
            raise ReleaseError(f"rebuild: {name!r} is not an image ({', '.join(sorted(images))})")
        if not image.built:
            raise ReleaseError(f"rebuild: {name} is not built by a Trenova release")
    return names, False


def previous_tag(git: Git, commit: str, version: str) -> str | None:
    older = [t for t in git.tags(commit) if TAG.fullmatch(t) and key(t[1:]) < key(version)]
    return max(older, key=lambda t: key(t[1:])) if older else None


def release_tags(version: str, all_tags: Iterable[str]) -> list[str]:
    """The tags this release gives every image: its version, and major.minor
    and latest only while no newer release has them, so re-running an old
    release never moves them back."""
    versions = [t[1:] for t in all_tags if TAG.fullmatch(t)] + [version]
    major, minor, _ = key(version)
    tags = [version]
    same_minor = [v for v in versions if key(v)[:2] == (major, minor)]
    if max(same_minor, key=key) == version:
        tags.append(f"{major}.{minor}")
    if max(versions, key=key) == version:
        tags.append("latest")
    return tags


def dig(doc: Any, path: str) -> Any:
    for part in path.split("."):
        if not isinstance(doc, dict) or part not in doc:
            return None
        doc = doc[part]
    return doc


def pinned_tag(values_text: str, image: Image) -> str:
    node = dig(yaml.safe_load(values_text), image.pinned)
    if not isinstance(node, dict) or node.get("repository") != image.repository:
        raise ReleaseError(f"{VALUES_FILE}: {image.pinned} does not pin {image.repository}")
    tag = node.get("tag")
    if not isinstance(tag, str) or not tag or "@" in tag:
        raise ReleaseError(f"{VALUES_FILE}: {image.pinned}.tag must be a plain tag")
    return tag


@dataclasses.dataclass
class Previous:
    version: str
    digest: str


def newest_published(registry: Registry, repository: str, at_most: str) -> str | None:
    tags = [t for t in registry.tags(repository) if VERSION.fullmatch(t) and key(t) <= key(at_most)]
    return max(tags, key=key) if tags else None


def find_previous(
    image: Image,
    prev_version: str,
    prev_tag: str,
    prev_manifest: dict[str, Any] | None,
    git: Git,
    registry: Registry,
) -> Previous | None:
    """What the previous release ran for an image: its manifest's entry, or,
    for a release before manifests, the digest of its tag on the registry."""
    if prev_manifest is not None:
        entry = prev_manifest["images"].get(image.name)
        if entry is None or entry.get("repository") != image.repository:
            return None
        return Previous(entry["version"], entry["digest"])
    if image.pinned:
        try:
            tag = pinned_tag(git.show(prev_tag, VALUES_FILE), image)
        except (ReleaseError, subprocess.CalledProcessError):
            return None
    elif image.lagging:
        tag = newest_published(registry, image.repository, prev_version)
        if tag is None:
            return None
    else:
        tag = prev_version
    digest = registry.digest(image.repository, tag)
    return Previous(tag, digest) if digest else None


def build_reason(
    image: Image,
    tag: str,
    prev_tag: str | None,
    prev: Previous | None,
    forced: set[str],
    decided: dict[str, dict[str, Any]],
    git: Git,
    registry: Registry,
) -> str:
    """Why an image is built, or "" when its previous digest is carried
    forward."""
    if prev_tag is None:
        return "the first release"
    if image.name in forced:
        return "the re-run asked for it"
    files = git.changed(prev_tag, tag, image.inputs)
    if files:
        shown = ", ".join(files[:5]) + (f" and {len(files) - 5} more" if len(files) > 5 else "")
        return f"its inputs changed since {prev_tag}: {shown}"
    rebuilt = [d for d in image.depends_on if decided[d]["action"] == BUILD]
    if rebuilt:
        return f"{', '.join(rebuilt)} is rebuilt"
    if prev is None:
        return f"{prev_tag} has no digest for it"
    if registry.digest(image.repository, prev.digest) is None:
        return f"{prev.digest} of {prev_tag} is no longer on the registry"
    return ""


def plan(
    *,
    tag: str,
    rebuild: str,
    images: dict[str, Image],
    git: Git,
    registry: Registry,
    releases: Releases,
) -> dict[str, Any]:
    if not TAG.fullmatch(tag):
        raise ReleaseError(f"{tag!r} is not a release tag such as v0.13.1")
    version = tag[1:]
    forced, refresh = parse_rebuild(rebuild, images)
    commit = git.commit(tag)
    prev_tag = previous_tag(git, commit, version)
    prev_version = prev_tag[1:] if prev_tag else None
    prev_manifest = releases.manifest(prev_tag) if prev_tag else None
    values_text = git.show(tag, VALUES_FILE)

    entries: dict[str, dict[str, Any]] = {}
    ordered = sorted(images.values(), key=lambda i: (bool(i.depends_on), i.name))
    for image in ordered:
        prev = (
            find_previous(image, prev_version, prev_tag, prev_manifest, git, registry)
            if prev_tag and prev_version
            else None
        )
        entry: dict[str, Any] = {
            "repository": image.repository,
            "previousDigest": prev.digest if prev else None,
        }
        if image.pinned:
            pin = pinned_tag(values_text, image)
            digest = registry.digest(image.repository, pin)
            if digest is None:
                raise ReleaseError(
                    f"{image.repository}:{pin}, which the chart pins, is not on the registry"
                )
            entry.update(action=PIN, version=pin, digest=digest, reason=f"the chart pins {pin}")
            entries[image.name] = entry
            continue
        reason = build_reason(image, tag, prev_tag, prev, forced, entries, git, registry)
        if reason:
            entry.update(action=BUILD, version=version, digest=None, reason=reason)
        else:
            assert prev is not None
            entry.update(
                action=REUSE,
                version=prev.version,
                digest=prev.digest,
                reason=f"unchanged since {prev.version}",
            )
        entry["stage"] = 2 if image.depends_on else 1
        entries[image.name] = entry

    migrations = True if prev_tag is None else bool(git.changed(prev_tag, tag, [MIGRATIONS]))
    result: dict[str, Any] = {
        "version": version,
        "tag": tag,
        "commit": commit,
        "previous": prev_version,
        "previousTag": prev_tag,
        "previousManifest": prev_manifest is not None,
        "migrations": migrations,
        "refreshPins": refresh,
        "tags": release_tags(version, git.all_tags()),
        "images": {name: entries[name] for name in sorted(entries)},
    }
    return result


def outputs(p: dict[str, Any]) -> dict[str, str]:
    images = p["images"]

    def names(pred: Callable[[dict[str, Any]], bool]) -> str:
        return json.dumps(sorted(n for n, e in images.items() if pred(e)), separators=(",", ":"))

    return {
        "version": p["version"],
        "build": names(lambda e: e["action"] == BUILD and e["stage"] == 1),
        "build_late": names(lambda e: e["action"] == BUILD and e["stage"] == 2),
        "reuse": names(lambda e: e["action"] == REUSE),
        "plan": json.dumps(p, separators=(",", ":"), sort_keys=True),
    }


def summary(p: dict[str, Any]) -> str:
    lines = [
        f"## Release {p['version']}",
        "",
        f"Previous release: {p['previous'] or 'none'}"
        + ("" if p["previous"] is None else
           " (with a manifest)" if p["previousManifest"] else " (no manifest: digests by its tags)"),
        f"Migrations: {'yes' if p['migrations'] else 'no'}",
        "",
        "| Image | Decision | Version | Why |",
        "|---|---|---|---|",
    ]
    for name, e in p["images"].items():
        lines.append(f"| {name} | {e['action']} | {e['version']} | {e['reason']} |")
    return "\n".join(lines) + "\n"


# The build of one image, and the aliases of the reused ones ------------------------


def resolved_ref(p: dict[str, Any], name: str, registry: Registry) -> str:
    entry = p["images"][name]
    digest = entry["digest"] or registry.digest(entry["repository"], p["version"])
    if not digest:
        raise ReleaseError(f"{entry['repository']}:{p['version']} is not on the registry")
    return f"{entry['repository']}:{entry['version']}@{digest}"


def image_outputs(p: dict[str, Any], image: Image, registry: Registry) -> dict[str, str]:
    entry = p["images"][image.name]
    if entry["action"] != BUILD:
        raise ReleaseError(f"{image.name} is not built by this release")
    args = list(image.build_args)
    args += [(arg, resolved_ref(p, dep, registry)) for arg, dep in image.image_args]
    return {
        "repository": image.repository,
        "dockerfile": image.dockerfile,
        "context": image.context,
        "target": image.target,
        "platforms": ",".join(image.platforms),
        "qemu": "true" if any(not pl.endswith("/amd64") for pl in image.platforms) else "false",
        "build_args": "\n".join(f"{k}={v}" for k, v in args),
        "tags": "\n".join(f"type=raw,value={t}" for t in p["tags"]),
        "version": p["version"],
        "commit": p["commit"],
        "cache_scope": f"{image.name}-release",
    }


def alias_commands(p: dict[str, Any]) -> list[list[str]]:
    commands = []
    for e in p["images"].values():
        if e["action"] != REUSE:
            continue
        command = ["docker", "buildx", "imagetools", "create"]
        for t in p["tags"]:
            command += ["--tag", f"{e['repository']}:{t}"]
        command.append(f"{e['repository']}@{e['digest']}")
        commands.append(command)
    return commands


# The manifest ---------------------------------------------------------------------


def build_manifest(p: dict[str, Any], registry: Registry) -> dict[str, Any]:
    images: dict[str, Any] = {}
    for name, e in p["images"].items():
        # Built and reused images carry this release's tag now; a pinned one
        # still has the tag the chart pins.
        tag = e["version"] if e["action"] == PIN else p["version"]
        digest = registry.digest(e["repository"], tag)
        if digest is None:
            raise ReleaseError(f"{e['repository']}:{tag} is not on the registry")
        if e["action"] != BUILD and digest != e["digest"]:
            raise ReleaseError(f"{e['repository']}:{tag} is {digest}, not the planned {e['digest']}")
        images[name] = {
            "repository": e["repository"],
            "version": e["version"],
            "digest": digest,
            "changed": e["previousDigest"] is None or e["previousDigest"] != digest,
        }
    manifest: dict[str, Any] = {
        "schemaVersion": SCHEMA_VERSION,
        "version": p["version"],
        "commit": p["commit"],
    }
    if p["previous"]:
        manifest["previous"] = p["previous"]
    manifest["migrations"] = p["migrations"]
    manifest["images"] = images
    validate_manifest(manifest)
    return manifest


def validate_manifest(m: Any, names: Iterable[str] | None = None) -> None:
    """The shape of contract section 2; raises ReleaseError on the first fault."""

    def fail(why: str) -> None:
        raise ReleaseError(f"{MANIFEST_NAME}: {why}")

    if not isinstance(m, dict):
        fail("not an object")
    if m.get("schemaVersion") != SCHEMA_VERSION or isinstance(m.get("schemaVersion"), bool):
        fail(f"schemaVersion is {m.get('schemaVersion')!r}, not {SCHEMA_VERSION}")
    if not isinstance(m.get("version"), str) or not VERSION.fullmatch(m["version"]):
        fail("version is not X.Y.Z")
    if not isinstance(m.get("commit"), str) or not COMMIT.fullmatch(m["commit"]):
        fail("commit is not 40 lowercase hex")
    if "previous" in m:
        if not isinstance(m["previous"], str) or not VERSION.fullmatch(m["previous"]):
            fail("previous is not X.Y.Z")
        if key(m["previous"]) >= key(m["version"]):
            fail("previous is not older than version")
    if not isinstance(m.get("migrations"), bool):
        fail("migrations is not a boolean")
    images = m.get("images")
    if not isinstance(images, dict) or not images:
        fail("images is empty")
    for name, e in images.items():
        if not isinstance(name, str) or not NAME.fullmatch(name):
            fail(f"{name!r} is not an image name")
        if not isinstance(e, dict):
            fail(f"images.{name} is not an object")
        if not isinstance(e.get("repository"), str) or not e["repository"]:
            fail(f"images.{name}.repository is empty")
        if not isinstance(e.get("version"), str) or not e["version"] or "@" in e["version"]:
            fail(f"images.{name}.version is not a tag")
        if not isinstance(e.get("digest"), str) or not DIGEST.fullmatch(e["digest"]):
            fail(f"images.{name}.digest is not sha256:<64 hex>")
        if not isinstance(e.get("changed"), bool):
            fail(f"images.{name}.changed is not a boolean")
    if names is not None:
        missing = sorted(set(names) - set(images))
        if missing:
            fail(f"no entry for {', '.join(missing)}")


# The charts -----------------------------------------------------------------------


def chart_images(values: Any) -> Iterable[dict[str, Any]]:
    if isinstance(values, dict):
        if "repository" in values and "tag" in values:
            yield values
        for child in values.values():
            yield from chart_images(child)
    elif isinstance(values, list):
        for child in values:
            yield from chart_images(child)


def parse_digests(text: str) -> dict[str, str]:
    doc = yaml.safe_load(text) or {}
    if not isinstance(doc, dict):
        raise ReleaseError(f"{DIGESTS_NAME} is not a mapping")
    return {str(k): str(v) for k, v in doc.items()}


def third_party_pins(
    values: Any,
    manifest: dict[str, Any],
    previous: dict[str, str],
    refresh: bool,
    resolve: Callable[[str], str],
) -> dict[str, str]:
    """The digest of every image a chart names that the manifest does not:
    the previous release's when it pinned the same repository:tag, so a
    floating upstream tag (redis:8) moving between two releases restarts
    nothing; resolved afresh when the tag changed, when there is no previous
    pin, or on a `rebuild: all` re-run."""
    ours = {e["repository"] for e in manifest["images"].values()}
    pins: dict[str, str] = {}
    for image in chart_images(values):
        repository, tag = image["repository"], image["tag"]
        if repository in ours or "@" in str(tag):
            continue
        if not tag:
            raise ReleaseError(f"{repository} names no tag and is not in the manifest")
        ref = f"{repository}:{tag}"
        digest = previous.get(ref) if not refresh else None
        if not digest:
            digest = resolve(ref)
        if not DIGEST.fullmatch(digest):
            raise ReleaseError(f"{ref}: no digest ({digest!r})")
        pins[ref] = digest
    return pins


def render_digests(pins: dict[str, str]) -> str:
    if not pins:
        return "{}\n"
    return "".join(f'"{ref}": "{digest}"\n' for ref, digest in sorted(pins.items()))


POSTGRES_BLOCK = re.compile(r'^postgres:\n(?:  .*\n|\n)*?    tag: "[^"]*"', re.M)


def point_postgres(values_text: str, version: str) -> str:
    block = POSTGRES_BLOCK.search(values_text)
    if not block:
        raise ReleaseError("postgres.image.tag not found in values.yaml")
    updated = re.sub(r'tag: "[^"]*"$', f'tag: "{version}"', block.group(0), flags=re.M)
    return values_text.replace(block.group(0), updated, 1)


def docker_digest(ref: str) -> str:
    return subprocess.run(
        ["docker", "buildx", "imagetools", "inspect", ref, "--format", "{{.Manifest.Digest}}"],
        check=True, capture_output=True, text=True,
    ).stdout.strip()


def pin_charts(
    manifest: dict[str, Any],
    charts_dir: Path,
    previous_dir: Path | None,
    refresh: bool,
    resolve: Callable[[str], str],
) -> dict[str, dict[str, str]]:
    written: dict[str, dict[str, str]] = {}
    postgres = manifest["images"].get("postgres")
    for chart in CHARTS:
        directory = charts_dir / chart
        values_path = directory / "values.yaml"
        text = values_path.read_text()
        if chart == "trenova" and postgres is not None:
            text = point_postgres(text, postgres["version"])
            values_path.write_text(text)
        previous: dict[str, str] = {}
        if previous_dir is not None and (previous_dir / chart / DIGESTS_NAME).is_file():
            previous = parse_digests((previous_dir / chart / DIGESTS_NAME).read_text())
        pins = third_party_pins(yaml.safe_load(text), manifest, previous, refresh, resolve)
        (directory / DIGESTS_NAME).write_text(render_digests(pins))
        (directory / MANIFEST_NAME).write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
        written[chart] = pins
    return written


# Dockerfile inputs ------------------------------------------------------------------


def _logical_lines(text: str) -> Iterable[str]:
    current = ""
    for raw in text.splitlines():
        line = raw.strip()
        if not current and (not line or line.startswith("#")):
            continue
        if line.endswith("\\"):
            current += line[:-1] + " "
            continue
        current += line
        yield current
        current = ""
    if current:
        yield current


def copied_sources(dockerfile_text: str) -> list[str]:
    """The context paths a Dockerfile's COPY and ADD lines read (never a
    --from= stage or image)."""
    sources: list[str] = []
    for line in _logical_lines(dockerfile_text):
        words = shlex.split(line, posix=True)
        if not words or words[0].upper() not in ("COPY", "ADD"):
            continue
        args = [w for w in words[1:] if not w.startswith("--")]
        if any(w.startswith("--from") for w in words[1:]):
            continue
        if args and args[0].startswith("["):
            args = json.loads(" ".join(args))
        if len(args) < 2:
            raise ReleaseError(f"cannot read {line!r}")
        sources.extend(args[:-1])
    return sources


def _covered(path: str, inputs: Iterable[str]) -> bool:
    """Whether a repository path is one of the inputs or inside one of the
    input directories."""
    for given in inputs:
        if given.endswith("/"):
            directory = given.rstrip("/")
            if path == directory or path.startswith(directory + "/"):
                return True
        elif path == given:
            return True
    return False


def _in_context(context: str, source: str) -> str:
    return os.path.normpath(os.path.join(context, source))


def _ignore_file(image: Image, exists: Callable[[str], bool]) -> str:
    """The dockerignore BuildKit applies: <Dockerfile>.dockerignore, else the
    context's .dockerignore, else none."""
    own = f"{image.dockerfile}.dockerignore"
    if exists(own):
        return own
    shared = os.path.normpath(os.path.join(image.context, ".dockerignore"))
    return shared if exists(shared) else ""


def _allowlist(ignore_text: str) -> list[str] | None:
    """The paths an allowlist dockerignore (`*`, then `!path` lines) lets in;
    None when it is not one."""
    lines = [line.strip() for line in ignore_text.splitlines()]
    lines = [line for line in lines if line and not line.startswith("#")]
    if not lines or lines[0] != "*":
        return None
    return [line[1:] for line in lines if line.startswith("!")]


def check_image_inputs(
    image: Image, read: Callable[[str], str], exists: Callable[[str], bool]
) -> list[str]:
    problems: list[str] = []
    for given in image.inputs:
        if not exists(given.rstrip("/")):
            problems.append(f"{image.name}: input {given} does not exist")
    ignore = _ignore_file(image, exists)
    if ignore and ignore not in image.inputs:
        problems.append(f"{image.name}: {ignore} decides its build context and is not an input")
    context = os.path.normpath(image.context)
    for source in copied_sources(read(image.dockerfile)):
        if any(ch in source for ch in "$*?["):
            problems.append(
                f"{image.name}: {image.dockerfile} copies {source!r}, which cannot be checked"
            )
            continue
        path = _in_context(image.context, source)
        if path != context:
            if not _covered(path, image.inputs):
                problems.append(
                    f"{image.name}: {image.dockerfile} copies {path}, which is not an input"
                )
            continue
        # The whole context: only an allowlist dockerignore makes what it
        # reads exact, and then every path it lets in is an input and every
        # input under the context is let in.
        allowed = _allowlist(read(ignore)) if ignore else None
        if allowed is None:
            problems.append(
                f"{image.name}: {image.dockerfile} copies its whole context ({image.context}); "
                f"give it {image.dockerfile}.dockerignore starting from `*` that lets in what it reads"
            )
            continue
        let_in = {_in_context(image.context, a) for a in allowed}
        declared = {
            given.rstrip("/")
            for given in image.inputs
            if context != "." and _covered(given.rstrip("/"), [context + "/"])
        }
        for missing in sorted(let_in - declared):
            problems.append(
                f"{image.name}: {ignore} lets {missing} into the build and it is not an input"
            )
        for extra in sorted(declared - let_in):
            problems.append(f"{image.name}: input {extra} is not let in by {ignore}")
    return problems


BUILD_STEP = "docker/build-push-action"


def workflow_dockerfiles(workflow_text: str) -> list[str]:
    """The Dockerfiles a workflow's build steps name (an expression names the
    list's own entry)."""
    doc = yaml.safe_load(workflow_text) or {}
    files = []
    for job in (doc.get("jobs") or {}).values():
        for step in job.get("steps") or []:
            if BUILD_STEP in str(step.get("uses", "")):
                files.append(str((step.get("with") or {}).get("file", "")))
    return files


def release_workflows(read: Callable[[str], str]) -> list[str]:
    seen, queue = [], [".github/workflows/release.yml"]
    while queue:
        path = queue.pop()
        if path in seen:
            continue
        seen.append(path)
        doc = yaml.safe_load(read(path)) or {}
        for job in (doc.get("jobs") or {}).values():
            uses = str(job.get("uses", ""))
            if uses.startswith("./.github/workflows/"):
                queue.append(uses[2:])
            for step in job.get("steps") or []:
                step_uses = str(step.get("uses", ""))
                if step_uses.startswith("./.github/actions/"):
                    queue.append(step_uses[2:].rstrip("/") + "/action.yml")
    return seen


def composite_dockerfiles(action_text: str) -> list[str]:
    doc = yaml.safe_load(action_text) or {}
    return [
        str((s.get("with") or {}).get("file", ""))
        for s in (doc.get("runs") or {}).get("steps") or []
        if BUILD_STEP in str(s.get("uses", ""))
    ]


def check_inputs(images: dict[str, Image], read: Callable[[str], str], exists: Callable[[str], bool]) -> list[str]:
    problems: list[str] = []
    for image in images.values():
        if not image.built:
            continue
        if not exists(image.dockerfile):
            problems.append(f"{image.name}: {image.dockerfile} does not exist")
            continue
        problems += check_image_inputs(image, read, exists)
    dockerfiles = {i.dockerfile.removeprefix("./") for i in images.values() if i.built}
    for path in release_workflows(read):
        text = read(path)
        named = composite_dockerfiles(text) if path.endswith("action.yml") else workflow_dockerfiles(text)
        for file in named:
            if "${{" in file:
                continue
            if file.removeprefix("./") not in dockerfiles:
                problems.append(f"{path} builds {file}, which {IMAGES_FILE} does not list")
    return problems


# The command line -----------------------------------------------------------------


def read_images() -> dict[str, Image]:
    return load_images((ROOT / IMAGES_FILE).read_text())


def ghcr() -> GHCR:
    user, token = os.environ.get("GHCR_USER", ""), os.environ.get("GHCR_TOKEN", "")
    if not user or not token:
        raise ReleaseError("GHCR_USER and GHCR_TOKEN are needed to read the registry")
    return GHCR(user, token)


def write_outputs(values: dict[str, str]) -> None:
    path = os.environ.get("GITHUB_OUTPUT")
    if not path:
        for k, v in values.items():
            print(f"{k}={v}")
        return
    with open(path, "a") as f:
        for k, v in values.items():
            delimiter = f"EOF_{os.urandom(8).hex()}"
            f.write(f"{k}<<{delimiter}\n{v}\n{delimiter}\n")


def write_summary(text: str) -> None:
    path = os.environ.get("GITHUB_STEP_SUMMARY")
    if path:
        with open(path, "a") as f:
            f.write(text)
    print(text, file=sys.stderr)


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("check-inputs")
    p_plan = sub.add_parser("plan")
    p_plan.add_argument("--tag", required=True)
    p_plan.add_argument("--rebuild", default="")
    p_plan.add_argument("--out", required=True)
    p_image = sub.add_parser("image")
    p_image.add_argument("name")
    p_image.add_argument("--plan", required=True)
    p_alias = sub.add_parser("aliases")
    p_alias.add_argument("--plan", required=True)
    p_alias.add_argument("--run", action="store_true", help="run the commands, not only print them")
    p_manifest = sub.add_parser("manifest")
    p_manifest.add_argument("--plan", required=True)
    p_manifest.add_argument("--out", required=True)
    p_pin = sub.add_parser("pin-charts")
    p_pin.add_argument("--manifest", required=True)
    p_pin.add_argument("--charts", default=str(ROOT / "deploy" / "helm"))
    p_pin.add_argument("--previous", default="")
    p_pin.add_argument("--refresh", action="store_true")
    p_validate = sub.add_parser("validate")
    p_validate.add_argument("file")
    args = parser.parse_args(argv)

    try:
        if args.command == "check-inputs":
            problems = check_inputs(
                read_images(),
                lambda p: (ROOT / p).read_text(),
                lambda p: (ROOT / p).exists(),
            )
            for problem in problems:
                print(f"::error::{problem}")
            if not problems:
                print(f"{IMAGES_FILE}: every image's inputs are what its Dockerfile reads")
            return 1 if problems else 0
        if args.command == "plan":
            result = plan(
                tag=args.tag,
                rebuild=args.rebuild,
                images=read_images(),
                git=LocalGit(),
                registry=ghcr(),
                releases=GitHubReleases(),
            )
            Path(args.out).write_text(json.dumps(result, indent=2, sort_keys=True) + "\n")
            write_outputs(outputs(result))
            write_summary(summary(result))
            return 0
        if args.command == "image":
            p = json.loads(Path(args.plan).read_text())
            images = read_images()
            if args.name not in images:
                raise ReleaseError(f"{args.name} is not in {IMAGES_FILE}")
            write_outputs(image_outputs(p, images[args.name], ghcr()))
            return 0
        if args.command == "aliases":
            p = json.loads(Path(args.plan).read_text())
            for command in alias_commands(p):
                print(" ".join(shlex.quote(c) for c in command), flush=True)
                if args.run:
                    subprocess.run(command, check=True)
            return 0
        if args.command == "manifest":
            p = json.loads(Path(args.plan).read_text())
            manifest = build_manifest(p, ghcr())
            validate_manifest(manifest, read_images())
            Path(args.out).write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
            print(json.dumps(manifest, indent=2, sort_keys=True))
            return 0
        if args.command == "pin-charts":
            manifest = json.loads(Path(args.manifest).read_text())
            validate_manifest(manifest, read_images())
            written = pin_charts(
                manifest,
                Path(args.charts),
                Path(args.previous) if args.previous else None,
                args.refresh,
                docker_digest,
            )
            for chart, pins in written.items():
                print(f"{chart}: {render_digests(pins)}", end="")
            return 0
        if args.command == "validate":
            validate_manifest(json.loads(Path(args.file).read_text()), read_images())
            print(f"{args.file}: a valid release manifest")
            return 0
    except ReleaseError as e:
        print(f"::error::{e}")
        return 1
    return 2


if __name__ == "__main__":
    sys.exit(main())
