"""JSON schemas for kubeconform from CustomResourceDefinitions.

    python3 tools/helm/crd-schemas.py crds.yaml schemas/

writes schemas/<group>/<kind>_<version>.json for every CRD in crds.yaml, with
additionalProperties false on every object that lists its properties, so
kubeconform -strict refuses a misspelt field in a custom resource as it does
in a built-in one. The chart's CI validates the CloudNativePG resources it
renders against the operator version it is tested with.
"""

import json, os, sys, yaml

def strict(node):
    if isinstance(node, dict):
        if node.get("type") == "object" and "properties" in node and "additionalProperties" not in node:
            node["additionalProperties"] = False
        for v in node.values():
            strict(v)
    elif isinstance(node, list):
        for v in node:
            strict(v)

out = sys.argv[2]
for doc in yaml.safe_load_all(open(sys.argv[1])):
    if not doc or doc.get("kind") != "CustomResourceDefinition":
        continue
    group = doc["spec"]["group"]
    kind = doc["spec"]["names"]["kind"].lower()
    for v in doc["spec"]["versions"]:
        schema = v["schema"]["openAPIV3Schema"]
        strict(schema)
        os.makedirs(os.path.join(out, group), exist_ok=True)
        with open(os.path.join(out, group, f"{kind}_{v['name']}.json"), "w") as f:
            json.dump(schema, f)
        print(group, kind, v["name"])
