"""Checks a release restarts only what it rebuilt (docs/engineering/releases.md).

    python3 -I tools/helm/check-release-renders.py deploy/helm/trenova deploy/helm/trenova-operator

Renders both charts as three published releases would, each a copy of the
chart at its version with its release manifest and third-party digests packaged
in, and compares the workloads (`helm` on PATH, or $HELM):

  0.13.0  every image built
  0.13.1  only tms and tms-db rebuilt (a TMS change with a migration)
  0.13.2  only the client rebuilt

From 0.13.0 to 0.13.1 every workload but the API and the worker renders the
same spec (the client, imaging, GTC, the embedding service, the data stores,
PostgreSQL's Cluster and ImageCatalog, the operator), and the release Job's
inputs change, so it runs. From 0.13.1 to 0.13.2 the TMS's pods, its
configuration and the release Job are unchanged, so Helm leaves them, and the
migration does not run again. A spec that is the same is what Kubernetes
compares: a resource's own labels may name the release, a pod template never.
"""

import json, os, shutil, subprocess, sys, tempfile, yaml

trenova_chart, operator_chart = sys.argv[1], sys.argv[2]
helm = os.environ.get("HELM", "helm")
errors = []

COMMIT = "0123456789abcdef0123456789abcdef01234567"
REPOS = {
    "tms": "ghcr.io/emoss08/trenova/tms",
    "tms-db": "ghcr.io/emoss08/trenova/tms-db",
    "client": "ghcr.io/emoss08/trenova/client",
    "imaging": "ghcr.io/emoss08/trenova/imaging",
    "operator": "ghcr.io/emoss08/trenova/operator",
    "tenant-agent": "ghcr.io/emoss08/trenova/tenant-agent",
    "postgres": "ghcr.io/emoss08/trenova/postgres",
    "export-runner": "ghcr.io/emoss08/trenova/export-runner",
    "gtc": "ghcr.io/emoss08/gtc",
    "embedding": "ghcr.io/emoss08/trenova/embedding",
}


def digest(seed):
    return "sha256:" + (seed.encode().hex() * 64)[:64]


def values_tag(path):
    node = yaml.safe_load(open(os.path.join(trenova_chart, "values.yaml")))
    for part in path.split("."):
        node = node[part]
    return node["tag"]


def manifest(version, previous, built, before=None):
    """A release that built `built` at `version` and carried the rest forward."""
    images = {}
    for name, repository in REPOS.items():
        if name == "gtc":
            v = values_tag("gtc.image")
        elif name == "embedding":
            v = values_tag("embedding.image")
        elif name in built:
            v = "0.11.5" if name == "postgres" else version
        else:
            v = before["images"][name]["version"]
        d = digest(f"{name}@{v}")
        images[name] = {
            "repository": repository,
            "version": v,
            "digest": d,
            "changed": before is None or before["images"][name]["digest"] != d,
        }
    m = {"schemaVersion": 1, "version": version, "commit": COMMIT, "migrations": "tms-db" in built,
         "images": images}
    if previous:
        m["previous"] = previous
    return m


def third_party(chart):
    pins = {}

    def walk(node):
        if isinstance(node, dict):
            if "repository" in node and "tag" in node:
                if node["repository"] not in REPOS.values() and "@" not in str(node["tag"]):
                    ref = f'{node["repository"]}:{node["tag"]}'
                    pins[ref] = digest(ref)
            for child in node.values():
                walk(child)

    walk(yaml.safe_load(open(os.path.join(chart, "values.yaml"))))
    return "".join(f'"{ref}": "{d}"\n' for ref, d in sorted(pins.items())) or "{}\n"


def publish(src, version, m, work):
    """The chart as the release workflow packages it."""
    out = os.path.join(work, f"{os.path.basename(src)}-{version}")
    shutil.copytree(src, out, symlinks=False)
    chart_yaml = os.path.join(out, "Chart.yaml")
    text = open(chart_yaml).read()
    text = text.replace("version: 0.0.0-dev", f"version: {version}", 1)
    text = text.replace("appVersion: 0.0.0-dev", f"appVersion: {version}", 1)
    open(chart_yaml, "w").write(text)
    values = os.path.join(out, "values.yaml")
    if src == trenova_chart:
        text = open(values).read()
        pg = m["images"]["postgres"]["version"]
        block_start = text.index("\npostgres:\n")
        tag_at = text.index('    tag: "', block_start)
        end = text.index('"', tag_at + len('    tag: "'))
        text = text[: tag_at + len('    tag: "')] + pg + text[end:]
        open(values, "w").write(text)
    open(os.path.join(out, "release-manifest.json"), "w").write(json.dumps(m, indent=2))
    open(os.path.join(out, "image-digests.yaml"), "w").write(third_party(out))
    return out


def render(chart, args):
    result = subprocess.run(
        [helm, "template", "trenova", chart, "-n", "trenova", *args],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        sys.exit(f"helm template {chart}: {result.stderr}")
    return [d for d in yaml.safe_load_all(result.stdout) if d]


WORKLOADS = {"Deployment", "StatefulSet", "Job", "Cluster", "ImageCatalog"}


def workloads(docs):
    return {
        (d["kind"], d["metadata"]["name"]): d
        for d in docs
        if d.get("kind") in WORKLOADS
    }


def config(docs):
    for d in docs:
        if d.get("kind") == "ConfigMap" and d["metadata"]["name"].endswith("-config"):
            return yaml.safe_load(d["data"]["config.yaml"])
    sys.exit("no TMS config rendered")


def component(d):
    return (d["metadata"].get("labels") or {}).get("app.kubernetes.io/component", "")


def inputs(d):
    return (d["metadata"].get("annotations") or {}).get("trenova.app/release-inputs")


def images(d):
    spec = d["spec"]
    if d["kind"] == "ImageCatalog":
        return [i["image"] for i in spec["images"]]
    if d["kind"] == "Cluster":
        return []
    pod = spec["template"]["spec"]
    return [c["image"] for c in (pod.get("initContainers") or []) + pod["containers"]]


base = ["--set", "domain=trenova.example.com", "--set", "storageDomain=storage.trenova.example.com"]
shapes = {
    "default": ["--set", "embedding.enabled=true"],
    "managed": ["-f", os.path.join(trenova_chart, "ci", "managed-values.yaml")],
}
TMS = {"api", "worker"}

with tempfile.TemporaryDirectory() as work:
    m0 = manifest("0.13.0", "0.12.1", set(REPOS) - {"gtc", "embedding"})
    m1 = manifest("0.13.1", "0.13.0", {"tms", "tms-db"}, m0)
    m2 = manifest("0.13.2", "0.13.1", {"client"}, m1)
    charts = {v: publish(trenova_chart, v, m, work) for v, m in
              (("0.13.0", m0), ("0.13.1", m1), ("0.13.2", m2))}

    for shape, args in shapes.items():
        r0, r1, r2 = (render(charts[v], base + args) for v in ("0.13.0", "0.13.1", "0.13.2"))
        w0, w1, w2 = workloads(r0), workloads(r1), workloads(r2)
        if set(w0) != set(w1) or set(w1) != set(w2):
            errors.append(f"{shape}: the releases render different workloads")
            continue
        seen = {component(d) for d in w0.values()}
        for need in ("api", "worker", "client", "imaging", "gtc", "release", "postgres"):
            if need not in seen:
                errors.append(f"{shape}: no {need} workload rendered")
        if shape == "default" and "embedding" not in seen:
            errors.append(f"{shape}: no embedding workload rendered")

        for key, d0 in w0.items():
            d1, d2 = w1[key], w2[key]
            part = component(d0)
            for d, m in ((d0, m0), (d1, m1), (d2, m2)):
                for image in images(d):
                    for e in m["images"].values():
                        if image.startswith(e["repository"] + ":") or image.startswith(e["repository"] + "@"):
                            want = f'{e["repository"]}:{e["version"]}@{e["digest"]}'
                            if image != want:
                                errors.append(f"{shape}: {key} runs {image}, not {want}")
            if part in TMS:
                if d0["spec"] == d1["spec"]:
                    errors.append(f"{shape}: {key} is unchanged by a release that rebuilt tms")
                if d1["spec"] != d2["spec"]:
                    errors.append(f"{shape}: {key} changes with a release that did not rebuild tms")
            elif part == "release":
                if inputs(d0) == inputs(d1):
                    errors.append(f"{shape}: the release Job does not run for a release that rebuilt tms-db")
                if d1["spec"] != d2["spec"] or inputs(d1) != inputs(d2) or d1["metadata"]["name"] != d2["metadata"]["name"]:
                    errors.append(f"{shape}: the release Job runs again for a release that did not rebuild tms-db")
            elif part == "client":
                if d0["spec"] != d1["spec"]:
                    errors.append(f"{shape}: {key} changes with a release that did not rebuild it")
                if d1["spec"] == d2["spec"]:
                    errors.append(f"{shape}: {key} is unchanged by a release that rebuilt it")
            elif d0["spec"] != d1["spec"] or d1["spec"] != d2["spec"]:
                errors.append(f"{shape}: {key} changes with a release that did not rebuild it")

        c0, c1, c2 = config(r0), config(r1), config(r2)
        if c0["app"]["version"] != "0.13.0" or c1["app"]["version"] != "0.13.1":
            errors.append(f"{shape}: app.version is not the tms image's version")
        if c1 != c2:
            errors.append(f"{shape}: the TMS configuration changes with a release that did not rebuild tms")

    # The operator's own pod: unchanged by 0.13.1, which did not rebuild it.
    ops = {v: publish(operator_chart, v, m, work) for v, m in (("0.13.0", m0), ("0.13.1", m1))}
    o0, o1 = (workloads(render(ops[v], [])) for v in ("0.13.0", "0.13.1"))
    for key, d0 in o0.items():
        if d0["spec"] != o1[key]["spec"]:
            errors.append(f"operator: {key} changes with a release that did not rebuild the operator")
        if images(d0) != [f'{REPOS["operator"]}:0.13.0@{digest("operator@0.13.0")}']:
            errors.append(f"operator: {key} runs {images(d0)}")

    # Without a manifest (an unreleased chart) images keep their tags.
    plain = workloads(render(trenova_chart, base))
    for key, d in plain.items():
        if component(d) in TMS and images(d)[-1] != f'{REPOS["tms"]}:0.0.0-dev':
            errors.append(f"unreleased: {key} runs {images(d)[-1]}")

for e in errors:
    print(f"::error::{e}")
if errors:
    sys.exit(1)
print("a release restarts only what it rebuilt")
