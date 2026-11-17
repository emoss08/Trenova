"""Checks every database a rendered chart makes asks for enough memory.

    python3 tools/helm/check-postgres-memory.py rendered.yaml [expected clusters]

CloudNativePG refuses a Cluster whose memory request is lower than its
shared_buffers, and the chart renders more than one: the live database and,
during a restore, a candidate beside it. Every Cluster in the file must ask
for at least its shared_buffers. With a number, the file must hold that many
Clusters, so a render that was meant to include a candidate cannot pass
without one.
"""

import re, sys, yaml

BINARY = {"": 1, "ki": 2**10, "mi": 2**20, "gi": 2**30, "ti": 2**40}
DECIMAL = {"k": 10**3, "m": 10**6, "g": 10**9, "t": 10**12}
POSTGRES = {"kb": 2**10, "mb": 2**20, "gb": 2**30, "tb": 2**40}


def quantity(text):
    m = re.fullmatch(r"([0-9.]+)([A-Za-z]*)", str(text).strip())
    if not m:
        raise ValueError(f"not a quantity: {text!r}")
    unit = m.group(2).lower()
    scale = BINARY.get(unit, DECIMAL.get(unit))
    if scale is None:
        raise ValueError(f"not a quantity: {text!r}")
    return float(m.group(1)) * scale


def shared_buffers(text):
    m = re.fullmatch(r"([0-9]+)\s*([A-Za-z]*)", str(text).strip())
    if not m:
        raise ValueError(f"not a PostgreSQL size: {text!r}")
    unit = m.group(2).lower()
    if unit == "":
        return int(m.group(1)) * 8 * 2**10
    if unit not in POSTGRES:
        raise ValueError(f"not a PostgreSQL size: {text!r}")
    return int(m.group(1)) * POSTGRES[unit]


path = sys.argv[1]
expected = int(sys.argv[2]) if len(sys.argv) > 2 else None
clusters = [
    d
    for d in yaml.safe_load_all(open(path))
    if d and d.get("kind") == "Cluster" and str(d.get("apiVersion", "")).startswith("postgresql.cnpg.io/")
]

errors = []
if expected is not None and len(clusters) != expected:
    errors.append(f"expected {expected} database clusters, found {len(clusters)}")
for c in clusters:
    name = c["metadata"]["name"]
    spec = c.get("spec") or {}
    buffers = ((spec.get("postgresql") or {}).get("parameters") or {}).get("shared_buffers")
    request = (((spec.get("resources") or {}).get("requests")) or {}).get("memory")
    if buffers is None:
        continue
    if request is None:
        errors.append(f"{name}: sets shared_buffers {buffers} and asks for no memory")
        continue
    if quantity(request) < shared_buffers(buffers):
        errors.append(f"{name}: asks for {request} of memory, less than its shared_buffers {buffers}")

for e in errors:
    print(f"::error::{e}")
sys.exit(1 if errors else 0)
