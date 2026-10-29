import collections
import sys

dg, srcfile = sys.argv[1], sys.argv[2]
nodes = {}
edges = collections.defaultdict(set)
for line in open(dg):
    p = line.rstrip("\n").split("\t")
    if p[0] == "N":
        nodes[p[1]] = p[2]
    else:
        edges[p[1]].add(p[2])
users = collections.defaultdict(set)
for a, bs in edges.items():
    for b in bs:
        if nodes.get(b) == srcfile and nodes.get(a) != srcfile:
            users[b].add(nodes[a])
by = collections.defaultdict(list)
for n, f in nodes.items():
    if f != srcfile:
        continue
    key = ",".join(sorted(users[n])) or "(none)"
    by[key].append(n)
for k in sorted(by):
    print(f"{k}\n    {' '.join(sorted(by[k]))[:300]}")
