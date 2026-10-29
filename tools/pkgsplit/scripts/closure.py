import collections
import sys

dg = sys.argv[1]
roots = sys.argv[2].split(",")
nodes = {}
edges = collections.defaultdict(set)
for line in open(dg):
    p = line.rstrip("\n").split("\t")
    if p[0] == "N":
        nodes[p[1]] = (p[2], int(p[3]))
    else:
        edges[p[1]].add(p[2])
seen = set()
stack = list(roots)
while stack:
    n = stack.pop()
    if n in seen:
        continue
    seen.add(n)
    stack.extend(edges[n])
byfile = collections.defaultdict(list)
for n in seen:
    byfile[nodes[n][0]].append(n)
total = sum(nodes[n][1] for n in seen)
print("closure", len(seen), "decls", total, "lines")
for f in sorted(byfile):
    names = sorted(byfile[f])
    print(f"{f:28} {sum(nodes[n][1] for n in names):5}  {' '.join(names)[:200]}")
