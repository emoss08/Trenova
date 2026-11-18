"""Checks CloudNativePG can reach every database a rendered chart may hold.

    python3 tools/helm/check-postgres-network.py rendered.yaml

The chart denies every connection a rule does not admit. CloudNativePG must
reach each database's pods on its status port to read it, stop it and start it
again, and that has to hold for a database the release has left and keeps,
which no template renders: it is the cluster a restore's switch left behind.
So one policy must select the release's database pods by a label every
database carries (not by one cluster's name) and admit CloudNativePG's
namespace, and every Cluster must give its pods that label.
"""

import sys, yaml

STATUS_PORT = 8000

docs = [d for d in yaml.safe_load_all(open(sys.argv[1])) if d]
policies = [d for d in docs if d.get("kind") == "NetworkPolicy"]
clusters = [
    d
    for d in docs
    if d.get("kind") == "Cluster" and str(d.get("apiVersion", "")).startswith("postgresql.cnpg.io/")
]

errors = []
if not policies:
    sys.exit(0)


def admits_other_namespace(policy):
    for rule in (policy.get("spec") or {}).get("ingress") or []:
        ports = [p.get("port") for p in rule.get("ports") or []]
        if STATUS_PORT not in ports:
            continue
        for peer in rule.get("from") or []:
            if (peer.get("namespaceSelector") or {}).get("matchLabels"):
                return True
    return False


selectors = []
for policy in policies:
    labels = (((policy.get("spec") or {}).get("podSelector")) or {}).get("matchLabels") or {}
    if "cnpg.io/cluster" in labels or not labels:
        continue
    if admits_other_namespace(policy):
        selectors.append(labels)

if not selectors:
    errors.append(
        "no NetworkPolicy admits CloudNativePG to the release's databases by a label they all carry: "
        "a database the release has left could not be stopped or gone back to"
    )

for cluster in clusters:
    name = cluster["metadata"]["name"]
    inherited = (((cluster.get("spec") or {}).get("inheritedMetadata")) or {}).get("labels") or {}
    if selectors and not any(all(inherited.get(k) == v for k, v in s.items()) for s in selectors):
        errors.append(f"{name}: its pods do not carry the labels the policy for every database selects")

for e in errors:
    print(f"::error::{e}")
sys.exit(1 if errors else 0)
