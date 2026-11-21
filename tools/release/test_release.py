"""Tests of tools/release/release.py with a fake git, registry and releases.

    python3 -m unittest discover -s tools/release -v
"""

import copy
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import release as r  # noqa: E402

ROOT = Path(__file__).resolve().parents[2]
IMAGES = r.load_images((ROOT / r.IMAGES_FILE).read_text())
COMMIT = "c" * 40

VALUES = """\
gtc:
  image:
    repository: ghcr.io/emoss08/gtc
    tag: 0.10.1
embedding:
  image:
    repository: ghcr.io/emoss08/trenova/embedding
    tag: nomic-embed-text-v1.5-ollama-0.40.5
"""


def d(seed: str) -> str:
    return "sha256:" + (seed.encode().hex() * 64)[:64]


class FakeGit:
    def __init__(self, tags, changed=(), values=VALUES, previous_values=VALUES):
        self.tags_ = list(tags)
        self.changed_ = list(changed)
        self.values = values
        self.previous_values = previous_values

    def commit(self, ref):
        return COMMIT

    def tags(self, commit):
        return list(self.tags_)

    def all_tags(self):
        return list(self.tags_)

    def changed(self, base, head, paths):
        return [f for f in self.changed_ if r._covered(f, paths)]

    def show(self, ref, path):
        assert path == r.VALUES_FILE
        return self.values if ref == "v0.13.1" else self.previous_values


class FakeRegistry:
    """Every image of every listed version is on the registry, as the
    release that built or aliased it left it, unless `missing` says not."""

    def __init__(self, versions=("0.13.0",), postgres=("0.11.0", "0.11.5"), missing=()):
        self.refs = {}
        for name, image in IMAGES.items():
            if image.pinned:
                continue
            for v in (postgres if name == "postgres" else versions):
                self.refs[(image.repository, v)] = d(f"{name}@{v}")
        self.refs[("ghcr.io/emoss08/gtc", "0.10.1")] = d("gtc@0.10.1")
        self.refs[("ghcr.io/emoss08/gtc", "0.10.0")] = d("gtc@0.10.0")
        self.refs[("ghcr.io/emoss08/trenova/embedding", "nomic-embed-text-v1.5-ollama-0.40.5")] = d("embedding")
        for key in missing:
            self.refs.pop(key, None)

    def digest(self, repository, reference):
        if reference.startswith("sha256:"):
            return reference if reference in {v for (repo, _), v in self.refs.items() if repo == repository} else None
        return self.refs.get((repository, reference))

    def tags(self, repository):
        return [t for (repo, t) in self.refs if repo == repository]

    def push(self, repository, tag, digest):
        self.refs[(repository, tag)] = digest


class FakeReleases:
    def __init__(self, manifests=None):
        self.manifests = manifests or {}

    def manifest(self, tag):
        return self.manifests.get(tag)


def previous_manifest(**overrides):
    """0.13.0's manifest, as a release with a manifest would have left it."""
    images = {}
    for name, image in IMAGES.items():
        version = {"postgres": "0.11.5", "gtc": "0.10.1",
                   "embedding": "nomic-embed-text-v1.5-ollama-0.40.5"}.get(name, "0.13.0")
        digest = d("embedding") if name == "embedding" else d(f"{name}@{version}")
        images[name] = {"repository": image.repository, "version": version,
                        "digest": digest, "changed": True}
    images.update(overrides)
    return {"schemaVersion": 1, "version": "0.13.0", "commit": "a" * 40,
            "previous": "0.12.1", "migrations": True, "images": images}


def run_plan(git, registry=None, releases=None, rebuild=""):
    return r.plan(tag="v0.13.1", rebuild=rebuild, images=IMAGES, git=git,
                  registry=registry or FakeRegistry(), releases=releases or FakeReleases())


def actions(p):
    return {n: e["action"] for n, e in p["images"].items()}


class PlanTest(unittest.TestCase):
    TAGS = ["v0.12.1", "v0.13.0", "v0.13.1"]

    def test_nothing_changed_reuses_every_image(self):
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(p["previous"], "0.13.0")
        self.assertTrue(p["previousManifest"])
        self.assertFalse(p["migrations"])
        for name, e in p["images"].items():
            want = r.PIN if IMAGES[name].pinned else r.REUSE
            self.assertEqual(e["action"], want, name)
        self.assertEqual(p["images"]["imaging"]["version"], "0.13.0")
        self.assertEqual(p["images"]["imaging"]["digest"], d("imaging@0.13.0"))
        self.assertEqual(p["images"]["postgres"]["version"], "0.11.5")
        out = r.outputs(p)
        self.assertEqual(out["build"], "[]")
        self.assertEqual(out["build_late"], "[]")

    def test_one_image_changed(self):
        git = FakeGit(self.TAGS, changed=["client/apps/web/src/main.tsx"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(actions(p)["client"], r.BUILD)
        self.assertIn("client/apps/web/src/main.tsx", p["images"]["client"]["reason"])
        self.assertEqual(p["images"]["client"]["version"], "0.13.1")
        self.assertIsNone(p["images"]["client"]["digest"])
        others = {n: a for n, a in actions(p).items() if n != "client" and not IMAGES[n].pinned}
        self.assertTrue(all(a == r.REUSE for a in others.values()), others)
        self.assertEqual(json.loads(r.outputs(p)["build"]), ["client"])

    def test_a_tms_change_builds_tms_and_tms_db_and_flags_migrations(self):
        git = FakeGit(self.TAGS, changed=["services/tms/schema/migrations/20261121_x.up.sql"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(json.loads(r.outputs(p)["build"]), ["tms", "tms-db"])
        self.assertTrue(p["migrations"])

    def test_a_tms_change_without_a_migration_is_not_flagged(self):
        git = FakeGit(self.TAGS, changed=["services/tms/internal/x.go"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(json.loads(r.outputs(p)["build"]), ["tms", "tms-db"])
        self.assertFalse(p["migrations"])

    def test_the_first_release_ever_builds_everything(self):
        p = run_plan(FakeGit(["v0.13.1"]), registry=FakeRegistry(versions=()))
        self.assertIsNone(p["previous"])
        self.assertTrue(p["migrations"])
        for name, e in p["images"].items():
            if IMAGES[name].pinned:
                self.assertEqual(e["action"], r.PIN)
                continue
            self.assertEqual(e["action"], r.BUILD, name)
            self.assertEqual(e["reason"], "the first release")
            self.assertIsNone(e["previousDigest"])
        self.assertEqual(json.loads(r.outputs(p)["build_late"]), ["export-runner"])

    def test_the_first_manifest_release_finds_digests_by_the_previous_tag(self):
        git = FakeGit(self.TAGS, changed=["services/tms/main.go"])
        p = run_plan(git)
        self.assertFalse(p["previousManifest"])
        self.assertEqual(actions(p)["imaging"], r.REUSE)
        self.assertEqual(p["images"]["imaging"]["version"], "0.13.0")
        self.assertEqual(p["images"]["imaging"]["digest"], d("imaging@0.13.0"))
        self.assertEqual(p["images"]["tms"]["previousDigest"], d("tms@0.13.0"))
        self.assertEqual(p["images"]["gtc"]["previousDigest"], d("gtc@0.10.1"))

    def test_postgres_keeps_its_newest_published_tag(self):
        registry = FakeRegistry(postgres=("0.9.0", "0.11.5", "0.11.10", "0.14.0"))
        p = run_plan(FakeGit(self.TAGS), registry=registry)
        pg = p["images"]["postgres"]
        self.assertEqual(pg["action"], r.REUSE)
        self.assertEqual(pg["version"], "0.11.10", "newest X.Y.Z that is not newer than 0.13.0")
        self.assertEqual(pg["digest"], d("postgres@0.11.10"))

    def test_postgres_with_no_published_image_is_built(self):
        registry = FakeRegistry(postgres=())
        p = run_plan(FakeGit(self.TAGS), registry=registry)
        self.assertEqual(actions(p)["postgres"], r.BUILD)
        self.assertEqual(actions(p)["export-runner"], r.BUILD)

    def test_a_rebuilt_postgres_rebuilds_the_export_runner(self):
        git = FakeGit(self.TAGS, changed=["deploy/Dockerfile.postgres"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(actions(p)["postgres"], r.BUILD)
        self.assertEqual(actions(p)["export-runner"], r.BUILD)
        self.assertEqual(p["images"]["export-runner"]["reason"], "postgres is rebuilt")
        out = r.outputs(p)
        self.assertEqual(json.loads(out["build"]), ["postgres"])
        self.assertEqual(json.loads(out["build_late"]), ["export-runner"])

    def test_the_export_runner_alone(self):
        git = FakeGit(self.TAGS, changed=["deploy/export-runner/export.sh"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        self.assertEqual(actions(p)["export-runner"], r.BUILD)
        self.assertEqual(actions(p)["postgres"], r.REUSE)

    def test_a_forced_rebuild(self):
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": previous_manifest()}),
                     rebuild="imaging, operator")
        self.assertEqual(json.loads(r.outputs(p)["build"]), ["imaging", "operator"])
        self.assertEqual(p["images"]["imaging"]["reason"], "the re-run asked for it")
        self.assertFalse(p["refreshPins"])

    def test_rebuild_all(self):
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": previous_manifest()}),
                     rebuild="all")
        built = {n for n, a in actions(p).items() if a == r.BUILD}
        self.assertEqual(built, {n for n, i in IMAGES.items() if i.built})
        self.assertTrue(p["refreshPins"])

    def test_a_rebuild_naming_no_image_is_refused(self):
        with self.assertRaisesRegex(r.ReleaseError, "not an image"):
            run_plan(FakeGit(self.TAGS), rebuild="imagin")
        with self.assertRaisesRegex(r.ReleaseError, "not built"):
            run_plan(FakeGit(self.TAGS), rebuild="gtc")
        with self.assertRaisesRegex(r.ReleaseError, "stands alone"):
            run_plan(FakeGit(self.TAGS), rebuild="all,tms")

    def test_a_missing_previous_digest_builds(self):
        m = previous_manifest()
        del m["images"]["tenant-agent"]
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": m}))
        self.assertEqual(actions(p)["tenant-agent"], r.BUILD)
        self.assertEqual(p["images"]["tenant-agent"]["reason"], "v0.13.0 has no digest for it")

        registry = FakeRegistry(missing=[("ghcr.io/emoss08/trenova/imaging", "0.13.0")])
        p = run_plan(FakeGit(self.TAGS), registry=registry)
        self.assertEqual(actions(p)["imaging"], r.BUILD, "no manifest and no tag: build")

    def test_a_previous_digest_gone_from_the_registry_builds(self):
        m = previous_manifest(client={"repository": IMAGES["client"].repository, "version": "0.12.0",
                                      "digest": d("gone"), "changed": False})
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": m}))
        self.assertEqual(actions(p)["client"], r.BUILD)
        self.assertIn("no longer on the registry", p["images"]["client"]["reason"])

    def test_a_carried_image_keeps_the_version_it_was_built_under(self):
        m = previous_manifest(imaging={"repository": IMAGES["imaging"].repository,
                                       "version": "0.11.2", "digest": d("imaging@0.13.0"),
                                       "changed": False})
        p = run_plan(FakeGit(self.TAGS), releases=FakeReleases({"v0.13.0": m}))
        self.assertEqual(p["images"]["imaging"]["version"], "0.11.2")

    def test_the_previous_release_is_the_newest_older_tag(self):
        git = FakeGit(["v0.9.0", "v0.13.0", "v0.12.1", "v0.13.1", "v0.13.1-rc1", "v0.14.0", "x"])
        p = run_plan(git)
        self.assertEqual(p["previous"], "0.13.0")
        self.assertEqual(p["tags"], ["0.13.1", "0.13"], "an older release never moves latest")

    def test_pinned_images_come_from_the_chart(self):
        values = VALUES.replace("tag: 0.10.1", "tag: 0.10.0")
        p = run_plan(FakeGit(self.TAGS, values=values), releases=FakeReleases({"v0.13.0": previous_manifest()}))
        gtc = p["images"]["gtc"]
        self.assertEqual((gtc["action"], gtc["version"], gtc["digest"]), (r.PIN, "0.10.0", d("gtc@0.10.0")))
        self.assertEqual(gtc["previousDigest"], d("gtc@0.10.1"))

    def test_a_pinned_image_missing_from_the_registry_stops_the_release(self):
        values = VALUES.replace("tag: 0.10.1", "tag: 0.99.0")
        with self.assertRaisesRegex(r.ReleaseError, "not on the registry"):
            run_plan(FakeGit(self.TAGS, values=values))

    def test_a_tag_that_is_not_a_release_is_refused(self):
        with self.assertRaisesRegex(r.ReleaseError, "release tag"):
            r.plan(tag="v0.13.1-rc1", rebuild="", images=IMAGES, git=FakeGit(self.TAGS),
                   registry=FakeRegistry(), releases=FakeReleases())


class TagsTest(unittest.TestCase):
    def test_the_newest_release_moves_latest_and_its_minor(self):
        self.assertEqual(r.release_tags("0.13.1", ["v0.13.0"]), ["0.13.1", "0.13", "latest"])

    def test_a_patch_of_an_older_minor_moves_only_its_minor(self):
        self.assertEqual(r.release_tags("0.12.3", ["v0.12.2", "v0.13.0"]), ["0.12.3", "0.12"])

    def test_re_running_an_old_patch_moves_nothing_shared(self):
        self.assertEqual(r.release_tags("0.12.1", ["v0.12.1", "v0.12.2", "v0.13.0"]), ["0.12.1"])


def built_registry(p, built_digests):
    """The registry once the build and alias jobs ran."""
    registry = FakeRegistry()
    for name, e in p["images"].items():
        if e["action"] == r.BUILD:
            registry.push(e["repository"], p["version"], built_digests[name])
        elif e["action"] == r.REUSE:
            registry.push(e["repository"], p["version"], e["digest"])
    return registry


class ManifestTest(unittest.TestCase):
    def plan_tms(self, **kw):
        git = FakeGit(PlanTest.TAGS, changed=["services/tms/internal/x.go",
                                              "services/tms/schema/migrations/1.sql"])
        return run_plan(git, **kw)

    def test_a_tms_release_after_a_release_without_a_manifest(self):
        p = self.plan_tms()
        registry = built_registry(p, {"tms": d("tms@0.13.1"), "tms-db": d("tms-db@0.13.1")})
        m = r.build_manifest(p, registry)
        r.validate_manifest(m, IMAGES)
        self.assertEqual(m["version"], "0.13.1")
        self.assertEqual(m["previous"], "0.13.0")
        self.assertEqual(m["commit"], COMMIT)
        self.assertTrue(m["migrations"])
        changed = {n for n, e in m["images"].items() if e["changed"]}
        self.assertEqual(changed, {"tms", "tms-db"})
        self.assertEqual(m["images"]["imaging"]["version"], "0.13.0")
        self.assertEqual(m["images"]["tms"]["version"], "0.13.1")
        self.assertEqual(m["images"]["postgres"]["version"], "0.11.5")
        self.assertEqual(m["images"]["gtc"]["version"], "0.10.1")

    def test_with_a_previous_manifest(self):
        p = self.plan_tms(releases=FakeReleases({"v0.13.0": previous_manifest()}))
        registry = built_registry(p, {"tms": d("tms@0.13.1"), "tms-db": d("tms-db@0.13.1")})
        m = r.build_manifest(p, registry)
        self.assertEqual({n for n, e in m["images"].items() if e["changed"]}, {"tms", "tms-db"})

    def test_the_first_release_changes_everything_and_has_no_previous(self):
        p = run_plan(FakeGit(["v0.13.1"]), registry=FakeRegistry(versions=()))
        built = {n: d(f"{n}@new") for n, e in p["images"].items() if e["action"] == r.BUILD}
        m = r.build_manifest(p, built_registry(p, built))
        self.assertNotIn("previous", m)
        self.assertTrue(all(e["changed"] for e in m["images"].values()))

    def test_a_rebuild_with_the_same_digest_is_not_a_change(self):
        p = run_plan(FakeGit(PlanTest.TAGS), rebuild="imaging",
                     releases=FakeReleases({"v0.13.0": previous_manifest()}))
        m = r.build_manifest(p, built_registry(p, {"imaging": d("imaging@0.13.0")}))
        self.assertFalse(m["images"]["imaging"]["changed"])

    def test_an_image_that_was_not_pushed_stops_the_manifest(self):
        p = self.plan_tms()
        registry = built_registry(p, {"tms": d("x"), "tms-db": d("y")})
        registry.refs.pop(("ghcr.io/emoss08/trenova/tms-db", "0.13.1"))
        with self.assertRaisesRegex(r.ReleaseError, "tms-db:0.13.1 is not on the registry"):
            r.build_manifest(p, registry)

    def test_an_alias_that_moved_stops_the_manifest(self):
        p = self.plan_tms()
        registry = built_registry(p, {"tms": d("x"), "tms-db": d("y")})
        registry.push(IMAGES["client"].repository, "0.13.1", d("someone else"))
        with self.assertRaisesRegex(r.ReleaseError, "not the planned"):
            r.build_manifest(p, registry)

    def test_validation(self):
        good = previous_manifest()
        r.validate_manifest(good, IMAGES)
        for mutate, why in [
            (lambda m: m.update(schemaVersion=2), "schemaVersion"),
            (lambda m: m.update(schemaVersion=True), "schemaVersion"),
            (lambda m: m.update(version="v0.13.0"), "version"),
            (lambda m: m.update(commit="ABC"), "commit"),
            (lambda m: m.update(previous="0.13.0"), "previous is not older"),
            (lambda m: m.update(migrations="yes"), "migrations"),
            (lambda m: m["images"]["tms"].update(digest="sha256:abc"), "digest"),
            (lambda m: m["images"]["tms"].update(changed=1), "changed"),
            (lambda m: m["images"]["tms"].update(version=""), "version"),
            (lambda m: m["images"].pop("gtc"), "no entry for gtc"),
        ]:
            bad = copy.deepcopy(good)
            mutate(bad)
            with self.assertRaisesRegex(r.ReleaseError, why):
                r.validate_manifest(bad, IMAGES)

    def test_readers_tolerate_more(self):
        m = previous_manifest()
        m["images"]["something-new"] = dict(m["images"]["tms"])
        m["unknown"] = True
        r.validate_manifest(m)


class BuildTest(unittest.TestCase):
    def test_the_export_runner_is_built_from_this_releases_postgres(self):
        git = FakeGit(PlanTest.TAGS, changed=["deploy/Dockerfile.postgres"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        registry = FakeRegistry()
        registry.push(IMAGES["postgres"].repository, "0.13.1", d("postgres@0.13.1"))
        out = r.image_outputs(p, IMAGES["export-runner"], registry)
        self.assertEqual(out["build_args"],
                         f"POSTGRES_IMAGE=ghcr.io/emoss08/trenova/postgres:0.13.1@{d('postgres@0.13.1')}")
        self.assertEqual(out["platforms"], "linux/amd64")
        self.assertEqual(out["qemu"], "false")

    def test_an_unchanged_postgres_is_the_carried_one(self):
        git = FakeGit(PlanTest.TAGS, changed=["deploy/export-runner/export.sh"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        out = r.image_outputs(p, IMAGES["export-runner"], FakeRegistry())
        self.assertIn(f"postgres:0.11.5@{d('postgres@0.11.5')}", out["build_args"])

    def test_build_parameters(self):
        p = run_plan(FakeGit(["v0.13.1"]), registry=FakeRegistry(versions=()))
        out = r.image_outputs(p, IMAGES["client"], FakeRegistry())
        self.assertEqual(out["build_args"], "VITE_API_URL=/api/v1")
        self.assertEqual(out["qemu"], "true")
        self.assertEqual(out["tags"].splitlines(),
                         ["type=raw,value=0.13.1", "type=raw,value=0.13", "type=raw,value=latest"])
        with self.assertRaisesRegex(r.ReleaseError, "not built"):
            r.image_outputs(p, IMAGES["gtc"], FakeRegistry())

    def test_aliases_tag_the_carried_digest(self):
        git = FakeGit(PlanTest.TAGS, changed=["services/tms/x.go"])
        p = run_plan(git, releases=FakeReleases({"v0.13.0": previous_manifest()}))
        commands = {c[-1]: c for c in r.alias_commands(p)}
        imaging = commands[f"ghcr.io/emoss08/trenova/imaging@{d('imaging@0.13.0')}"]
        self.assertEqual(imaging[:4], ["docker", "buildx", "imagetools", "create"])
        self.assertEqual([imaging[i + 1] for i, a in enumerate(imaging) if a == "--tag"], [
            "ghcr.io/emoss08/trenova/imaging:0.13.1",
            "ghcr.io/emoss08/trenova/imaging:0.13",
            "ghcr.io/emoss08/trenova/imaging:latest",
        ])
        self.assertFalse(any("tms@" in k for k in commands), "a built image is not aliased")
        self.assertFalse(any("gtc" in k for k in commands), "a pinned image is not ours to tag")


class ChartsTest(unittest.TestCase):
    MANIFEST = previous_manifest()

    def test_third_party_pins_carry_forward(self):
        values = {"redis": {"image": {"repository": "redis", "tag": "8"}},
                  "minio": {"image": {"repository": "x/silo", "tag": "R@sha256:" + "1" * 64}},
                  "tms": {"image": {"repository": "ghcr.io/emoss08/trenova/tms", "tag": ""}},
                  "temporal": {"image": {"repository": "temporalio/auto-setup", "tag": "1.29.8"}}}
        previous = {"redis:8": d("redis-old"), "temporalio/auto-setup:1.29.7": d("t-old")}
        resolved = []

        def resolve(ref):
            resolved.append(ref)
            return d(ref)

        pins = r.third_party_pins(values, self.MANIFEST, previous, False, resolve)
        self.assertEqual(pins, {"redis:8": d("redis-old"), "temporalio/auto-setup:1.29.8": d("temporalio/auto-setup:1.29.8")})
        self.assertEqual(resolved, ["temporalio/auto-setup:1.29.8"], "only a new tag is looked up")

        pins = r.third_party_pins(values, self.MANIFEST, previous, True, resolve)
        self.assertEqual(pins["redis:8"], d("redis:8"), "rebuild: all looks every tag up again")

    def test_pin_charts_writes_both_charts(self):
        with tempfile.TemporaryDirectory() as tmp:
            charts = Path(tmp) / "charts"
            for chart in r.CHARTS:
                (charts / chart).mkdir(parents=True)
            (charts / "trenova" / "values.yaml").write_text(
                'redis:\n  image:\n    repository: redis\n    tag: "8"\n'
                'postgres:\n  # the image\n  image:\n    repository: ghcr.io/emoss08/trenova/postgres\n    tag: "0.11.0"\n'
            )
            (charts / "trenova-operator" / "values.yaml").write_text(
                'image:\n  repository: ghcr.io/emoss08/trenova/operator\n  tag: ""\n'
            )
            written = r.pin_charts(self.MANIFEST, charts, None, False, lambda ref: d(ref))
            self.assertEqual(written["trenova-operator"], {})
            self.assertEqual((charts / "trenova-operator" / r.DIGESTS_NAME).read_text(), "{}\n")
            self.assertEqual((charts / "trenova" / r.DIGESTS_NAME).read_text(), f'"redis:8": "{d("redis:8")}"\n')
            self.assertIn('tag: "0.11.5"', (charts / "trenova" / "values.yaml").read_text())
            for chart in r.CHARTS:
                packaged = json.loads((charts / chart / r.MANIFEST_NAME).read_text())
                self.assertEqual(packaged, self.MANIFEST)


class InputsTest(unittest.TestCase):
    def files(self, extra=None):
        files = {
            "deploy/Dockerfile.a": "FROM x AS b\nCOPY --from=b /x /y\nCOPY a/go.mod \\\n  a/go.sum ./\nCOPY a/ ./a/\n",
            ".dockerignore": ".git\n",
            "a": "", "a/go.mod": "", "a/go.sum": "",
            "deploy/Dockerfile.w": "COPY . .\nCOPY vendor/ vendor/\n",
            "deploy/Dockerfile.w.dockerignore": "# allowlist\n*\n!src\n!vendor\n**/target\n",
            "w/src": "", "w/vendor": "",
            ".github/workflows/release.yml": "jobs:\n  b:\n    steps:\n      - uses: docker/build-push-action@x\n        with:\n          file: ${{ steps.p.outputs.dockerfile }}\n",
        }
        files.update(extra or {})
        return files

    def check(self, images_yaml, files):
        images = r.load_images(images_yaml)
        return r.check_inputs(images, lambda p: files[p], lambda p: p in files)

    GOOD = """images:
  a: {repository: r/a, dockerfile: deploy/Dockerfile.a, context: ., platforms: [x], inputs: [a/, deploy/Dockerfile.a, .dockerignore]}
  w: {repository: r/w, dockerfile: deploy/Dockerfile.w, context: w, platforms: [x], inputs: [w/src, w/vendor, deploy/Dockerfile.w, deploy/Dockerfile.w.dockerignore]}
"""

    def test_the_real_tree(self):
        problems = r.check_inputs(IMAGES, lambda p: (ROOT / p).read_text(), lambda p: (ROOT / p).exists())
        self.assertEqual(problems, [])

    def test_exact_inputs_pass(self):
        self.assertEqual(self.check(self.GOOD, self.files()), [])

    def test_a_copied_path_that_is_not_an_input(self):
        files = self.files({"deploy/Dockerfile.a": "COPY a/ ./a/\nCOPY b/x.txt /x\n", "b/x.txt": ""})
        problems = self.check(self.GOOD, files)
        self.assertEqual(problems, ["a: deploy/Dockerfile.a copies b/x.txt, which is not an input"])

    def test_the_dockerignore_is_an_input(self):
        images = self.GOOD.replace(", .dockerignore]", "]")
        self.assertIn("a: .dockerignore decides its build context and is not an input",
                      self.check(images, self.files()))

    def test_a_whole_context_needs_an_allowlist(self):
        files = self.files({"deploy/Dockerfile.w.dockerignore": "target\n"})
        self.assertTrue(any("copies its whole context" in p for p in self.check(self.GOOD, files)))

    def test_the_allowlist_and_the_inputs_agree(self):
        files = self.files({"deploy/Dockerfile.w.dockerignore": "*\n!src\n!vendor\n!scripts\n", "w/scripts": ""})
        self.assertIn("w: deploy/Dockerfile.w.dockerignore lets w/scripts into the build and it is not an input",
                      self.check(self.GOOD, files))
        files = self.files({"deploy/Dockerfile.w.dockerignore": "*\n!src\n"})
        self.assertIn("w: input w/vendor is not let in by deploy/Dockerfile.w.dockerignore",
                      self.check(self.GOOD, files))

    def test_a_workflow_building_an_unlisted_image(self):
        files = self.files({".github/workflows/release.yml": (
            "jobs:\n  b:\n    steps:\n      - uses: docker/build-push-action@x\n        with:\n          file: ./deploy/Dockerfile.z\n"
            "  c:\n    uses: ./.github/workflows/other.yml\n"
        ), ".github/workflows/other.yml": (
            "jobs:\n  d:\n    steps:\n      - uses: ./.github/actions/img\n"
        ), ".github/actions/img/action.yml": (
            "runs:\n  steps:\n    - uses: docker/build-push-action@x\n      with:\n        file: deploy/Dockerfile.y\n"
        )})
        problems = self.check(self.GOOD, files)
        self.assertIn(".github/workflows/release.yml builds ./deploy/Dockerfile.z, which deploy/release/images.yaml does not list", problems)
        self.assertIn(".github/actions/img/action.yml builds deploy/Dockerfile.y, which deploy/release/images.yaml does not list", problems)

    def test_a_missing_input(self):
        images = self.GOOD.replace("[a/, ", "[a/, nothing/, ")
        self.assertIn("a: input nothing/ does not exist", self.check(images, self.files()))

    def test_the_list_must_name_the_dockerfile(self):
        with self.assertRaisesRegex(r.ReleaseError, "Dockerfile"):
            r.load_images(self.GOOD.replace("inputs: [a/, deploy/Dockerfile.a, .dockerignore]", "inputs: [a/]"))


if __name__ == "__main__":
    unittest.main()
