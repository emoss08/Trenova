"""Checks a restart of a part leaves the rest of the instance as it was.

    python3 tools/helm/check-restarts.py rendered.yaml

The Web app's Caddy proxies the API, so its rollout keeps a ready pod and the
old pod pauses before it stops while the ingress drops its endpoint: without
the pause every API caller sees errors for a few seconds. The search indexing
service is ready only while its sinks and replication stream are (/readiness);
its liveness stays on /health so it is never restarted for a sink it cannot
reach. Temporal advertises loopback for its membership, so a restarted pod
never waits on the gone pod's address and crashes once, and it is ready only
once matching answers.
"""

import sys, yaml

PAUSE = 5

docs = [d for d in yaml.safe_load_all(open(sys.argv[1])) if d]
errors = []


def deployment(component):
    for d in docs:
        labels = (d.get("metadata") or {}).get("labels") or {}
        if d.get("kind") == "Deployment" and labels.get("app.kubernetes.io/component") == component:
            return d
    return None


def container(d, name):
    for c in d["spec"]["template"]["spec"].get("containers") or []:
        if c["name"] == name:
            return c
    return None


client = deployment("client")
if client:
    rolling = (client["spec"].get("strategy") or {}).get("rollingUpdate") or {}
    if (client["spec"].get("strategy") or {}).get("type") != "RollingUpdate" or str(rolling.get("maxUnavailable")) != "0":
        errors.append("the client's rollout may stop its only ready pod: it needs RollingUpdate with maxUnavailable 0")
    caddy = container(client, "caddy") or {}
    pause = ((((caddy.get("lifecycle") or {}).get("preStop")) or {}).get("sleep") or {}).get("seconds") or 0
    if pause < PAUSE:
        errors.append(f"the client stops before the ingress drops it: it needs a preStop sleep of at least {PAUSE}s")
    grace = client["spec"]["template"]["spec"].get("terminationGracePeriodSeconds", 30)
    if grace <= pause:
        errors.append("the client's terminationGracePeriodSeconds leaves Caddy no time to drain after its pause")

gtc = deployment("gtc")
if gtc:
    c = container(gtc, "gtc") or {}
    ready = ((c.get("readinessProbe") or {}).get("httpGet") or {}).get("path")
    live = ((c.get("livenessProbe") or {}).get("httpGet") or {}).get("path")
    if ready != "/readiness":
        errors.append(f"gtc's readiness probe is {ready!r}: only /readiness reflects its sinks")
    if live != "/health":
        errors.append(f"gtc's liveness probe is {live!r}: it must stay on /health so a down sink never restarts it")

temporal = deployment("temporal")
if temporal:
    c = container(temporal, "temporal") or {}
    env = {e["name"]: e for e in c.get("env") or []}
    broadcast = env.get("TEMPORAL_BROADCAST_ADDRESS") or {}
    if broadcast.get("value") != "127.0.0.1":
        errors.append("Temporal must advertise 127.0.0.1: a pod address makes a restart wait on the gone pod and crash")
    if temporal["spec"]["replicas"] > 1:
        errors.append("Temporal advertises loopback, which holds only for one replica")
    probe = " ".join(((c.get("readinessProbe") or {}).get("exec") or {}).get("command") or [])
    if "task-queue describe" not in probe:
        errors.append("Temporal's readiness probe does not ask matching: it says ready while matching has no hosts")

for e in errors:
    print(f"::error::{e}")
sys.exit(1 if errors else 0)
