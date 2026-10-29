import collections
import sys

sys.setrecursionlimit(100000)
dg = sys.argv[1]
core_roots = sys.argv[2].split(",")
nodes = {}
edges = collections.defaultdict(set)
for line in open(dg):
    p = line.rstrip("\n").split("\t")
    if p[0] == "N":
        nodes[p[1]] = (p[2], int(p[3]))
    else:
        edges[p[1]].add(p[2])
core = set()
stack = list(core_roots)
while stack:
    n = stack.pop()
    if n in core:
        continue
    core.add(n)
    stack.extend(edges[n])
rest = [n for n in nodes if n not in core]
index = {}
low = {}
on = set()
st = []
sccs = []
counter = [0]


def strong(v):
    index[v] = low[v] = counter[0]
    counter[0] += 1
    st.append(v)
    on.add(v)
    for w in edges[v]:
        if w in core:
            continue
        if w not in index:
            strong(w)
            low[v] = min(low[v], low[w])
        elif w in on:
            low[v] = min(low[v], index[w])
    if low[v] == index[v]:
        comp = []
        while True:
            w = st.pop()
            on.discard(w)
            comp.append(w)
            if w == v:
                break
        sccs.append(comp)


for v in rest:
    if v not in index:
        strong(v)
big = [c for c in sccs if len({nodes[n][0] for n in c}) > 1]
print("non-core sccs spanning files:", len(big))
for c in big:
    files = collections.Counter(nodes[n][0] for n in c)
    print(sum(nodes[n][1] for n in c), dict(files))
