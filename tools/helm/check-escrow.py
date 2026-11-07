"""Checks where a rendered chart reads the four escrowed keys from.

    python3 tools/helm/check-escrow.py rendered.yaml <escrow secret> <app secret>

Every TMS container must read TRENOVA_SECURITY_ENCRYPTION_KEY,
TRENOVA_DATABASE_RLS_SCOPEKEYID, TRENOVA_DATABASE_RLS_SCOPEKEY and
TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER from <escrow secret>. When that is not
the app Secret, the app Secret the chart generates must not hold those keys;
when it is, it must hold all four.
"""

import sys, yaml

ENV = {
    "TRENOVA_SECURITY_ENCRYPTION_KEY": "encryptionKey",
    "TRENOVA_DATABASE_RLS_SCOPEKEYID": "rlsScopeKeyId",
    "TRENOVA_DATABASE_RLS_SCOPEKEY": "rlsScopeKey",
    "TRENOVA_SECURITY_MFA_RECOVERYCODEPEPPER": "mfaRecoveryCodePepper",
}

path, escrow, app = sys.argv[1:4]
docs = [d for d in yaml.safe_load_all(open(path)) if d]


def pod_spec(doc):
    spec = doc.get("spec") or {}
    if doc["kind"] == "Pod":
        return spec
    return ((spec.get("template") or {}).get("spec")) or {}


errors, seen = [], set()
for doc in docs:
    if doc["kind"] not in ("Deployment", "StatefulSet", "Job", "Pod"):
        continue
    spec = pod_spec(doc)
    for c in spec.get("initContainers", []) + spec.get("containers", []):
        for e in c.get("env") or []:
            if e["name"] not in ENV:
                continue
            ref = (e.get("valueFrom") or {}).get("secretKeyRef") or {}
            where = f'{doc["kind"]}/{doc["metadata"]["name"]} {c["name"]} {e["name"]}'
            if ref.get("name") != escrow or ref.get("key") != ENV[e["name"]]:
                errors.append(f"{where} reads {ref.get('name')}/{ref.get('key')}, not {escrow}/{ENV[e['name']]}")
            seen.add(e["name"])

for name in sorted(set(ENV) - seen):
    errors.append(f"no container reads {name}")

secrets = [d for d in docs if d["kind"] == "Secret" and d["metadata"]["name"] == app]
if not secrets:
    errors.append(f"the app Secret {app} is not rendered")
for s in secrets:
    held = set(ENV.values()) & set((s.get("data") or {}).keys())
    if escrow != app and held:
        errors.append(f"the app Secret {app} still holds {sorted(held)}")
    if escrow == app and held != set(ENV.values()):
        errors.append(f"the app Secret {app} lacks {sorted(set(ENV.values()) - held)}")

for e in errors:
    print(f"::error::{path}: {e}")
if errors:
    sys.exit(1)
print(f"{path}: the escrowed keys are read from {escrow}")
