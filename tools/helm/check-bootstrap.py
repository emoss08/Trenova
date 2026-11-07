"""Checks how a rendered chart's release Job bootstraps the first administrator.

    python3 tools/helm/check-bootstrap.py rendered.yaml <password|link>

The bootstrap container must pass TRENOVA_BOOTSTRAP_ADMIN_SETUP as given. With
password it reads TRENOVA_BOOTSTRAP_ADMIN_PASSWORD from the app Secret; with
link no container in the Job is handed a bootstrap password at all.
"""

import sys, yaml

path, setup = sys.argv[1:3]
docs = [d for d in yaml.safe_load_all(open(path)) if d]

errors, found = [], False
for doc in docs:
    if doc["kind"] != "Job":
        continue
    spec = ((doc.get("spec") or {}).get("template") or {}).get("spec") or {}
    for c in spec.get("initContainers", []) + spec.get("containers", []):
        env = {e["name"]: e for e in c.get("env") or []}
        where = f'Job/{doc["metadata"]["name"]} {c["name"]}'
        if setup == "link" and "TRENOVA_BOOTSTRAP_ADMIN_PASSWORD" in env:
            errors.append(f"{where} is handed a bootstrap password in link mode")
        if c["name"] != "bootstrap":
            continue
        found = True
        got = (env.get("TRENOVA_BOOTSTRAP_ADMIN_SETUP") or {}).get("value")
        if got != setup:
            errors.append(f"{where} passes TRENOVA_BOOTSTRAP_ADMIN_SETUP={got!r}, not {setup!r}")
        if setup == "password" and "TRENOVA_BOOTSTRAP_ADMIN_PASSWORD" not in env:
            errors.append(f"{where} has no bootstrap password in password mode")

if not found:
    errors.append("no release Job with a bootstrap container is rendered")

for e in errors:
    print(f"::error::{e}")
if errors:
    sys.exit(1)
print(f"{path}: the first administrator is bootstrapped for a {setup}")
