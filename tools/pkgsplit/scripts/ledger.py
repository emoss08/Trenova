import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", "..", "..")) + "/"
WORK = os.environ.get("PKGSPLIT_WORK", "/tmp/pkgsplit") + "/"
BIN = os.environ.get("PKGSPLIT_BIN", WORK + "bin") + "/"
TMS = REPO + "services/tms/"
MOD = "github.com/emoss08/trenova/"
LEDGER = TMS + "docs/modules/symbol-moves.tsv"


def exported(rel):
    out = subprocess.run([BIN + "declgraph", TMS + rel], capture_output=True, text=True, check=True).stdout
    names = []
    for line in out.splitlines():
        p = line.split("\t")
        if p[0] == "N" and p[1][:1].isupper():
            names.append(p[1])
    return names


old_rel = sys.argv[1]
pairs = []
for new_rel in sys.argv[2:]:
    src, dst = new_rel.split(":") if ":" in new_rel else (old_rel, new_rel)
    for name in exported(dst):
        pairs.append(f"{MOD}{src}.{name}\t{MOD}{dst}.{name}")
target = {p.split("\t")[0]: p.split("\t")[1] for p in pairs}
lines = open(LEDGER).read().rstrip("\n").split("\n")
repointed = 0
for i, line in enumerate(lines):
    a, _, b = line.partition("\t")
    if b in target:
        lines[i] = a + "\t" + target[b]
        repointed += 1
existing = set(lines)
new = [p for p in pairs if p not in existing]
open(LEDGER, "w").write("\n".join(lines + new) + "\n")
print("repointed", repointed, "appended", len(new))
