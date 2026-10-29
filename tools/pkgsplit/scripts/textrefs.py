import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(HERE, "..", "..", "..")) + "/"
WORK = os.environ.get("PKGSPLIT_WORK", "/tmp/pkgsplit") + "/"
BIN = os.environ.get("PKGSPLIT_BIN", WORK + "bin") + "/"
MOD = "github.com/emoss08/trenova/"
LEDGER = "services/tms/docs/modules/symbol-moves.tsv"

old_rel = sys.argv[1]
old_name = os.path.basename(old_rel)
owner = {}
for rel in sys.argv[2:]:
    out = subprocess.run([BIN + "declgraph", REPO + "services/tms/" + rel], capture_output=True, text=True, check=True).stdout
    for line in out.splitlines():
        p = line.split("\t")
        if p[0] == "N" and p[1][:1].isupper():
            owner[p[1]] = rel
full = re.compile(re.escape(MOD + old_rel) + r"\.([A-Z][A-Za-z0-9_]*)")
short = re.compile(r"\b" + re.escape(old_name) + r"\.([A-Z][A-Za-z0-9_]*)")
targets = []
for root in ("services/tms", "docs", ".claude", "AGENTS.md", "CLAUDE.md"):
    base = REPO + root
    if os.path.isfile(base):
        targets.append(base)
        continue
    for d, dirs, files in os.walk(base):
        dirs[:] = [x for x in dirs if x not in ("node_modules", ".git", "worktrees")]
        for f in files:
            if f.endswith((".yml", ".yaml", ".md", ".tsv", ".graphqls")):
                targets.append(os.path.join(d, f))
changed = 0
for p in targets:
    if p.endswith(LEDGER):
        continue
    s = open(p).read()
    n = full.sub(lambda m: MOD + owner[m.group(1)] + "." + m.group(1) if m.group(1) in owner else m.group(0), s)
    n = short.sub(lambda m: os.path.basename(owner[m.group(1)]) + "." + m.group(1) if m.group(1) in owner else m.group(0), n)
    if n != s:
        open(p, "w").write(n)
        changed += 1
        print(p[len(REPO):])
print("changed", changed, file=sys.stderr)
