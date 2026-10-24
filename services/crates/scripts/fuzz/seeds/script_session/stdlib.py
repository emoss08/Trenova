import json, datetime, re, collections, itertools, math
d = collections.defaultdict(list)
for k, g in itertools.groupby(sorted(["a","b","a"])):
    d[k] = len(list(g))
print(json.dumps(d), datetime.date.today(), re.sub("a", "b", "aa"), math.floor(2.5))
