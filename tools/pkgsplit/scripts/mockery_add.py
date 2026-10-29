import re
import sys

path = sys.argv[1]
prefix = sys.argv[2]
names = sys.argv[3:]
src = open(path).read()
head, _, body = src.partition("packages:\n")
entries = re.findall(r"(  (\S+):\n(?:    .*\n)+)", body)
tail = body[sum(len(e[0]) for e in entries):]
assert "".join(e[0] for e in entries) == body[: len(body) - len(tail)], "unexpected layout"
have = {e[1]: e[0] for e in entries}
for n in names:
    key = prefix + n
    have.setdefault(key, "  " + key + ":\n    config:\n      all: true\n")
out = head + "packages:\n" + "".join(have[k] for k in sorted(have)) + tail
open(path, "w").write(out)
print("entries", len(have))
