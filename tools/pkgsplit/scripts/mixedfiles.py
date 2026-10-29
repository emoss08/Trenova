import collections
import sys

dg, roots = sys.argv[1], sys.argv[2].split(",")
nodes = {}
lines = {}
edges = collections.defaultdict(set)
for line in open(dg):
    p = line.rstrip("\n").split("\t")
    if p[0] == "N":
        nodes[p[1]] = p[2]
        lines[p[1]] = int(p[3])
    else:
        edges[p[1]].add(p[2])
core = set()
stack = list(roots)
while stack:
    n = stack.pop()
    if n in core:
        continue
    core.add(n)
    stack.extend(edges[n])
per = collections.defaultdict(lambda: [0, 0, [], []])
for n, f in nodes.items():
    i = 0 if n in core else 1
    per[f][i] += lines[n]
    per[f][2 + i].append(n)
for f, (c, o, cn, on) in sorted(per.items()):
    kind = "CORE" if o == 0 else ("OUT" if c == 0 else "MIXED")
    extra = f"  out: {' '.join(sorted(on))[:160]}" if kind == "MIXED" else ""
    print(f"{kind:5} {f:34} core={c:5} out={o:5}{extra}")
