import re,sys
def key(l):
    s=l.strip()
    if not s or s.startswith('//'): return None
    return s.split()[0]
def keelopts(l):
    m=re.search(r'keel:"([^"]*)"',l); return m.group(1).split(',') if m else None
def merge_line(o,t):
    ko,kt=keelopts(o),keelopts(t)
    if kt is None: return o
    if ko is None:
        # add theirs keel tag onto ours line
        return re.sub(r'`\s*$', '', o.rstrip()).rstrip('`') + ' keel:"%s"`'%','.join(kt) if o.rstrip().endswith('`') else o
    opts=list(ko)
    for x in kt:
        if x not in opts: opts.append(x)
    return re.sub(r'keel:"[^"]*"','keel:"%s"'%','.join(opts),o,1)
for p in sys.argv[1:]:
    lines=open(p).read().split('\n'); out=[]; i=0
    while i<len(lines):
        if lines[i].startswith('<<<<<<<'):
            ours=[];base=[];theirs=[];cur=ours;i+=1
            while not lines[i].startswith('>>>>>>>'):
                if lines[i].startswith('|||||||'): cur=base
                elif lines[i].startswith('======='): cur=theirs
                else: cur.append(lines[i])
                i+=1
            i+=1
            bk={key(l) for l in base if key(l)}
            tk={key(l):l for l in theirs if key(l)}
            ok={key(l) for l in ours if key(l)}
            res=[]
            for l in ours:
                k=key(l)
                res.append(merge_line(l,tk[k]) if k and k in tk else l)
            for l in theirs:
                k=key(l)
                if k and k not in ok and k not in bk: res.append(l)
            # theirs-only comment lines (doc comments) before struct: keep if not in ours
            for l in theirs:
                if not key(l) and l.strip().startswith('//') and l not in ours and l not in base: res.insert(0,l)
            out.extend(res)
        else:
            out.append(lines[i]); i+=1
    open(p,'w').write('\n'.join(out))
