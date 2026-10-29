import re,sys
pat=re.compile(r'^import \(\n(.*?)\n\)',re.S|re.M)
def key(line):
    m=re.search(r'"([^"]+)"',line); return (m.group(1) if m else "", line)
def isstd(line):
    m=re.search(r'"([^"]+)"',line)
    return m and "." not in m.group(1).split("/")[0]
n=0
for path in sys.argv[1:]:
    try: src=open(path).read()
    except FileNotFoundError: continue
    m=pat.search(src)
    if not m: continue
    body=m.group(1)
    if "/*" in body: continue
    lines=[l.strip() for l in body.split("\n") if l.strip()]
    if any(l.startswith("//") for l in lines): continue
    std=sorted({l for l in lines if isstd(l)},key=key)
    oth=sorted({l for l in lines if not isstd(l)},key=key)
    groups=["\n".join("\t"+l for l in g) for g in (std,oth) if g]
    new="import (\n"+"\n\n".join(groups)+"\n)"
    if new!=m.group(0):
        src=src[:m.start()]+new+src[m.end():]; open(path,"w").write(src); n+=1
print("normalized",n,file=sys.stderr)
