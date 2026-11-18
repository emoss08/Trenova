"""Checks a rendered chart serves every hostname it is given and links on one.

    python3 tools/helm/check-aliases.py rendered.yaml <domain> <primary> [alias ...]

An instance is served on domain and on every alias. The Ingress must route
each and list each among its TLS hosts, the client's Caddy must answer each,
the TMS must admit each one's origin for CORS and for its cross-origin check,
and the bundled MinIO (when it runs) must admit each origin. Every link the
TMS builds (the web app, problem types, password resets, the driver portal,
delivery links) must use the primary hostname, and the readiness probe stays
on domain.
"""

import sys, yaml

rendered, domain, primary, *aliases = sys.argv[1:]
hosts = [domain] + aliases
origins = [f"https://{h}" for h in hosts]
primary_origin = f"https://{primary}"

docs = [d for d in yaml.safe_load_all(open(rendered)) if d]
errors = []


def named(kind, suffix):
    return [d for d in docs if d.get("kind") == kind and d["metadata"]["name"].endswith(suffix)]


ingresses = [d for d in docs if d.get("kind") == "Ingress"]
if not ingresses:
    errors.append("no Ingress was rendered")
for ing in ingresses:
    spec = ing.get("spec") or {}
    ruled = {r.get("host") for r in spec.get("rules") or []}
    tls = {h for t in spec.get("tls") or [] for h in t.get("hosts") or []}
    for h in hosts:
        if h not in ruled:
            errors.append(f"Ingress {ing['metadata']['name']}: no rule for {h}")
        if spec.get("tls") and h not in tls:
            errors.append(f"Ingress {ing['metadata']['name']}: {h} is not a TLS host")

configs = [d for d in named("ConfigMap", "-config") if "config.yaml" in (d.get("data") or {})]
if not configs:
    errors.append("no ConfigMap holds the TMS's config.yaml")
for cm in configs:
    config = yaml.safe_load(cm["data"]["config.yaml"])
    lists = {
        "server.cors.allowedOrigins": ((config.get("server") or {}).get("cors") or {}).get("allowedOrigins") or [],
        "security.csrf.trustedOrigins": ((config.get("security") or {}).get("csrf") or {}).get("trustedOrigins") or [],
    }
    for key, got in lists.items():
        for o in origins:
            if o not in got:
                errors.append(f"{key} does not admit {o}")
    links = {
        "app.webBaseUrl": (config.get("app") or {}).get("webBaseUrl"),
        "app.problemTypeBaseUri": (config.get("app") or {}).get("problemTypeBaseUri"),
        "security.passwordReset.baseUrl": ((config.get("security") or {}).get("passwordReset") or {}).get("baseUrl"),
        "portal.baseUrl": (config.get("portal") or {}).get("baseUrl"),
        "reporting.deliveryLinkBaseUrl": (config.get("reporting") or {}).get("deliveryLinkBaseUrl"),
    }
    for key, got in links.items():
        if not str(got or "").startswith(primary_origin):
            errors.append(f"{key} is {got!r}, not on the primary {primary_origin}")


def container_env(deployment, container):
    for c in deployment["spec"]["template"]["spec"]["containers"]:
        if c["name"] == container:
            return c, {e["name"]: e.get("value") for e in c.get("env") or []}
    return None, {}


clients = named("Deployment", "-client")
if not clients:
    errors.append("no client Deployment was rendered")
for dep in clients:
    caddy, env = container_env(dep, "caddy")
    sites = [s.strip() for s in (env.get("DOMAIN") or "").split(",")]
    for h in hosts:
        if f"http://{h}:8080" not in sites:
            errors.append(f"the client's Caddy does not answer {h} (DOMAIN={env.get('DOMAIN')!r})")
    probe_host = [
        h.get("value")
        for h in (((caddy or {}).get("readinessProbe") or {}).get("httpGet") or {}).get("httpHeaders") or []
        if h.get("name") == "Host"
    ]
    if probe_host != [domain]:
        errors.append(f"the client's readiness probe asks for {probe_host}, not {domain}")

for dep in named("StatefulSet", "-minio"):
    _, env = container_env(dep, "minio")
    allowed = [o.strip() for o in (env.get("MINIO_API_CORS_ALLOW_ORIGIN") or "").split(",")]
    for o in origins:
        if o not in allowed:
            errors.append(f"the bundled MinIO does not admit {o}")

for e in errors:
    print(f"::error::{e}")
sys.exit(1 if errors else 0)
