import collections
import sys

dg, mapfile, core = sys.argv[1], sys.argv[2], sys.argv[3]
mixed = set(sys.argv[4:])
nodes = {}
edges = collections.defaultdict(set)
for line in open(dg):
    p = line.rstrip("\n").split("\t")
    if p[0] == "N":
        nodes[p[1]] = p[2]
    else:
        edges[p[1]].add(p[2])
filegroup = {}
for line in open(mapfile):
    p = line.split()
    if len(p) == 2:
        filegroup[p[0]] = p[1]
users = collections.defaultdict(set)
for a, bs in edges.items():
    for b in bs:
        if a != b:
            users[b].add(a)
assign = {n: filegroup.get(f, core) for n, f in nodes.items() if f not in mixed}
pending = [n for n, f in nodes.items() if f in mixed]


def step(force):
    changed = False
    for n in pending:
        if n in assign:
            continue
        mixed_refs = [t for t in edges[n] if t != n and nodes.get(t) in mixed]
        if len(mixed_refs) == 1 and len(edges[n]) <= 2:
            t = mixed_refs[0]
            if t in assign:
                assign[n] = assign[t]
                changed = True
            continue
        us = users[n]
        if any(u not in assign for u in us) and not force:
            continue
        gs = {assign[u] for u in us if u in assign}
        if len(gs) == 1:
            assign[n] = gs.pop()
        elif len(gs) > 1:
            assign[n] = core
        else:
            rg = {assign[t] for t in edges[n] if t in assign and assign[t] != core}
            if any(t not in assign for t in edges[n] if t != n) and not force:
                continue
            assign[n] = rg.pop() if len(rg) == 1 else core
        changed = True
    return changed


while step(False) or step(True):
    if all(n in assign for n in pending):
        break
for n in pending:
    assign.setdefault(n, core)
moves = collections.defaultdict(list)
for n in pending:
    if assign[n] != core:
        moves[(nodes[n], assign[n])].append(n)
for (f, g), names in sorted(moves.items()):
    print(f"SPLIT\t{f}\t{g}_{f}\t{g}\t{' '.join(sorted(names))}")
kept = sorted(n for n in pending if assign[n] == core)
print(f"KEEP\t{len(kept)}\t{' '.join(kept)[:400]}", file=sys.stderr)
