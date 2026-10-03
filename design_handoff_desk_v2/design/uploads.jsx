const MAX_ATTACHMENTS = 5;
const MAX_MB = 25;
const OK_EXT = ["pdf", "png", "jpg", "jpeg", "heic", "tif", "tiff", "csv", "xlsx", "xls", "docx", "txt", "eml"];
const extOf = n => (n.split(".").pop() || "").toLowerCase();
const fmtSize = b => b < 1024 * 1024 ? Math.max(1, Math.round(b / 1024)) + " KB" : (b / 1024 / 1024).toFixed(1) + " MB";
const FT = { pdf: ["PDF", "pdf"], png: ["PNG", "img"], jpg: ["JPG", "img"], jpeg: ["JPG", "img"], heic: ["HEIC", "img"], tif: ["TIFF", "img"], tiff: ["TIFF", "img"], csv: ["CSV", "sheet"], xlsx: ["XLSX", "sheet"], xls: ["XLS", "sheet"], docx: ["DOCX", "doc"], txt: ["TXT", "doc"], eml: ["EML", "doc"] };

const SAMPLES = [
  { name: "rate-con-acme-77812.pdf", size: 412000 },
  { name: "BOL-778.pdf", size: 1240000 },
  { name: "pod-photo.jpg", size: 2900000 },
];
const BAD_SAMPLES = [
  { name: "carrier-packet.zip", size: 8000000 },
  { name: "scan-batch-sept.pdf", size: 61000000 },
  { name: "locked-invoice.pdf", size: 320000 },
];

function useUploads() {
  const [atts, setAtts] = React.useState([]);
  const [note, setNote] = React.useState(null);
  const timers = React.useRef([]);
  React.useEffect(() => () => timers.current.forEach(clearInterval), []);
  const patch = (id, p) => setAtts(xs => xs.map(x => x.id === id ? { ...x, ...p } : x));
  const start = (a) => {
    const total = 600 + Math.min(2600, a.size / 1500);
    const t0 = performance.now();
    const iv = setInterval(() => {
      const p = Math.min(1, (performance.now() - t0) / total);
      if (a.failAt && p >= a.failAt) { clearInterval(iv); patch(a.id, { status: "error", progress: p, err: "Upload failed · connection dropped", retry: true }); return; }
      if (p >= 1) { clearInterval(iv); patch(a.id, a.locked ? { status: "error", progress: 1, err: "Password-protected · Desk can't open it" } : { status: "ready", progress: 1, documentId: "doc_" + a.id }); return; }
      patch(a.id, { progress: p });
    }, 60);
    timers.current.push(iv);
  };
  const add = (files) => {
    const list = Array.from(files);
    setAtts(cur => {
      const room = MAX_ATTACHMENTS - cur.length;
      if (list.length > room) setNote(`Up to ${MAX_ATTACHMENTS} files per message · ${list.length - Math.max(0, room)} not added`);
      else setNote(null);
      const next = list.slice(0, Math.max(0, room)).map((f, i) => {
        const ext = extOf(f.name);
        const a = { id: Math.random().toString(36).slice(2, 8), name: f.name, size: f.size, ext, status: "uploading", progress: 0, url: f.url || (f instanceof Blob && /^image\//.test(f.type) ? URL.createObjectURL(f) : null) };
        if (!OK_EXT.includes(ext)) return { ...a, status: "error", err: "Desk can't read ." + ext + " files" };
        if (f.size > MAX_MB * 1024 * 1024) return { ...a, status: "error", err: "Too large · max " + MAX_MB + " MB" };
        if (/locked/i.test(f.name)) a.locked = true;
        if (f.failOnce) a.failAt = 0.55;
        setTimeout(() => start(a), 40 + i * 120);
        return a;
      });
      return [...cur, ...next];
    });
  };
  const remove = id => { setAtts(xs => xs.filter(x => x.id !== id)); setNote(null); };
  const retry = id => setAtts(xs => xs.map(x => { if (x.id !== id) return x; const a = { ...x, status: "uploading", progress: 0, err: null, failAt: null, retry: false }; setTimeout(() => start(a), 40); return a; }));
  const clear = () => { setAtts([]); setNote(null); };
  const scanTimers = React.useRef({});
  const finishScan = id => { clearInterval(scanTimers.current[id]); setAtts(xs => xs.map(x => x.id === id ? { ...x, status: x.pages ? "ready" : "error", err: x.pages ? null : "No pages came through", name: x.pages ? `Scan ${new Date().toLocaleTimeString([], { hour: "numeric", minute: "2-digit" }).replace(/\s/g, "")} · ${x.pages} page${x.pages > 1 ? "s" : ""}.pdf` : x.name, ext: "pdf", size: (x.pages || 1) * 180000, documentId: "doc_" + id } : x)); };
  const scan = ({ device, total = 4 }) => {
    if (atts.length >= MAX_ATTACHMENTS) { setNote(`Up to ${MAX_ATTACHMENTS} files per message`); return; }
    const id = Math.random().toString(36).slice(2, 8);
    setAtts(xs => [...xs, { id, name: "Scanning on " + device, ext: "scan", size: 0, status: "scanning", pages: 0, device, progress: 0 }]);
    let n = 0;
    setTimeout(() => {
      scanTimers.current[id] = setInterval(() => { n++; setAtts(xs => xs.map(x => x.id === id ? { ...x, pages: n } : x)); if (n >= total) finishScan(id); }, 900);
      timers.current.push(scanTimers.current[id]);
    }, 1400);
    return id;
  };
  return { atts, add, remove: id => { clearInterval(scanTimers.current[id]); remove(id); }, retry, clear, note, setNote, scan, finishScan };
}

function FileIcon({ a, s = 32 }) {
  const [lab, kind] = FT[a.ext] || [a.ext.toUpperCase().slice(0, 4) || "FILE", "bad"];
  if (kind === "img" && a.url) return <span className="fi img" style={{ width: s, height: s, backgroundImage: `url(${a.url})` }}></span>;
  return <span className={"fi k-" + kind} style={{ width: s, height: s }}><span>{lab}</span></span>;
}

function Ring({ p, s = 18 }) {
  const r = s / 2 - 2, c = 2 * Math.PI * r;
  return <svg className="ring" width={s} height={s} viewBox={`0 0 ${s} ${s}`}><circle cx={s / 2} cy={s / 2} r={r} fill="none" stroke="currentColor" strokeOpacity=".2" strokeWidth="2"></circle><circle cx={s / 2} cy={s / 2} r={r} fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - p)} transform={`rotate(-90 ${s / 2} ${s / 2})`}></circle></svg>;
}

function FileChip({ a, onRemove, onRetry, onDone }) {
  if (a.status === "scanning") return (
    <div className="fp s-scanning" data-tip={a.pages ? "Pages are arriving from " + a.device : "Waiting for " + a.device + " · put the pages in the scanner"}>
      <span className="fp-scan"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4 15h16v4H4zM6 15V9l4-4h8v10"></path><path d="M8 18h.01"></path></svg></span>
      <span className="fp-n">{a.pages ? "Scanning" : "Waiting for scanner"}</span>
      <span className="fp-pages">{Array.from({ length: Math.min(a.pages, 4) }, (_, i) => <i key={i} style={{ animationDelay: "0ms" }}></i>)}</span>
      {a.pages > 0 && <span className="fp-pc">{a.pages + (a.pages > 1 ? " pages" : " page")}</span>}
      {a.pages > 0 && onDone && <button className="fp-b done" onClick={() => onDone(a.id)} title="Done scanning"><Ic n="check" s={11} w={2.4} /></button>}
      <button className="fp-b" onClick={() => onRemove(a.id)} title="Cancel scan"><Ic n="x" s={10} w={2.4} /></button>
    </div>
  );
  const tip = a.status === "error" ? a.err : a.status === "uploading" ? "Uploading · " + Math.round(a.progress * 100) + "%" : a.name + " · " + fmtSize(a.size);
  return (
    <div className={"fp s-" + a.status} data-tip={tip} style={{ "--p": a.progress || 0 }}>
      <FileIcon a={a} s={18} />
      <span className="fp-n">{a.name}</span>
      {a.status === "uploading" && <span className="fp-pc">{Math.round(a.progress * 100)}%</span>}
      {a.status === "error" && <span className="fp-e">{a.err.split(" · ")[0]}</span>}
      {a.status === "error" && a.retry && <button className="fp-b" onClick={() => onRetry(a.id)} title="Try again"><Ic n="replay" s={11} /></button>}
      <button className="fp-b" onClick={() => onRemove(a.id)} title="Remove"><Ic n="x" s={10} w={2.4} /></button>
    </div>
  );
}

function AttachRow({ up, lead }) {
  const [more, setMore] = React.useState(false);
  if (!up.atts.length && !up.note && !lead) return null;
  const SHOW = 3;
  const errs = up.atts.filter(a => a.status === "error").length;
  const rest = up.atts.slice(SHOW);
  return (
    <div className="ar">
      <div className="ar-l">
        {lead}
        {up.atts.slice(0, SHOW).map(a => <FileChip key={a.id} a={a} onRemove={up.remove} onRetry={up.retry} onDone={up.finishScan} />)}
        {rest.length > 0 && <span className="fp-more-w"><button className={"fp more" + (rest.some(a => a.status === "error") ? " has-err" : "")} onClick={() => setMore(m => !m)}>{`+${rest.length} more`}</button>
          {more && <div className="fp-pop" onMouseLeave={() => setMore(false)}>{rest.map(a => <FileChip key={a.id} a={a} onRemove={up.remove} onRetry={up.retry} onDone={up.finishScan} />)}</div>}</span>}
        {(up.note || errs > 0) && <span className="ar-note">{up.note ? up.note : errs === 1 ? "Retry or remove it to send with this file" : errs + " files can't be sent · retry or remove them"}</span>}
      </div>
    </div>
  );
}

const CAPTURE_DEVICES = [
  { id: "dev1", name: "AVERY-LAPTOP", online: true, sources: ["Fujitsu fi-8170", "Epson DS-530 II"] },
  { id: "dev2", name: "FRONT-DESK-PC", online: false, sources: ["Brother ADS-2700W"] },
];
const CAPTURE_PROFILES = ["Organization default", "BOL · 300 dpi · grayscale · both sides", "POD · color · split on blank page"];
const CAPTURE_TYPES = ["Let Desk decide", "Bill of lading", "Proof of delivery", "Rate confirmation", "Lumper receipt"];

function CapturePanel({ up, onBack, onClose, mode = "ok" }) {
  const devices = mode === "none" ? [] : mode === "offline" ? CAPTURE_DEVICES.map(d => ({ ...d, online: false })) : CAPTURE_DEVICES;
  const pref = (window.DESK_CFG || {}).scanDevice;
  const [dev, setDev] = React.useState((devices.find(x => x.id === pref) || devices[0] || {}).id);
  const [src, setSrc] = React.useState("default");
  const [prof, setProf] = React.useState(CAPTURE_PROFILES.includes((window.DESK_CFG || {}).scanProfile) ? window.DESK_CFG.scanProfile : CAPTURE_PROFILES[0]);
  const [type, setType] = React.useState(CAPTURE_TYPES[0]);
  const d = devices.find(x => x.id === dev);
  const start = () => { up.scan({ device: d.name }); onClose(); };
  return (
    <div className="cap">
      <div className="cap-h">
        <button className="ib" onClick={onBack} title="Back"><Ic n="chevL" s={13} /></button>
        <span><b>Scan from Capture</b><em>Pages scan on your computer and attach to this message</em></span>
      </div>
      {devices.length === 0 ? (
        <div className="cap-empty">
          <span className="cap-eic"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round"><path d="M4 15h16v4H4zM6 15V9l4-4h8v10"></path></svg></span>
          <b>No computer is set up to scan for you</b>
          <span>Install Trenova Capture, sign in from its tray icon, and approve the code it shows.</span>
          <div className="cap-acts"><button className="ec-btn ink">Download Trenova Capture</button><button className="ec-btn">My scanners</button></div>
        </div>
      ) : <>
        <div className="cap-f">
          <div className="cap-l">Computer</div>
          <div className="cap-devs">{devices.map(x => (
            <button key={x.id} className={"cap-dev" + (dev === x.id ? " on" : "")} onClick={() => { setDev(x.id); setSrc("default"); }}>
              <span className={"cap-dot" + (x.online ? " ok" : "")}></span><b>{x.name}</b><em>{x.online ? "Connected" : "Not connected"}</em>
              {dev === x.id && <Ic n="check" s={12} w={2.4} />}
            </button>))}</div>
          {d && !d.online && <div className="cap-warn"><Ic n="alert" s={12} w={2} />{d.name + " isn't connected. The scan waits a few minutes for it to come online."}</div>}
        </div>
        <div className="cap-grid">
          <label className="cap-sel"><span>Scanner</span><select value={src} onChange={e => setSrc(e.target.value)}><option value="default">Computer's default</option>{d && d.sources.map(s => <option key={s}>{s}</option>)}</select></label>
          <label className="cap-sel"><span>Scan settings</span><select value={prof} onChange={e => setProf(e.target.value)}>{CAPTURE_PROFILES.map(s => <option key={s}>{s}</option>)}</select></label>
          <label className="cap-sel wide"><span>Document type</span><select value={type} onChange={e => setType(e.target.value)}>{CAPTURE_TYPES.map(s => <option key={s}>{s}</option>)}</select></label>
        </div>
        <div className="cap-foot"><span>{d ? (d.online ? "Put the pages in the scanner after you start." : "Starts when " + d.name + " connects.") : ""}</span><button className="ec-btn ink" onClick={start}>Start scan</button></div>
      </>}
    </div>
  );
}

function AttachMenu({ up, onClose, startCapture }) {
  const inp = React.useRef(null);
  const [cap, setCap] = React.useState(startCapture || null);
  const root = React.useRef(null);
  React.useEffect(() => {
    const k = e => e.key === "Escape" && onClose();
    const off = e => root.current && !root.current.contains(e.target) && !e.target.closest(".cmp-b .ib") && onClose();
    window.addEventListener("keydown", k); document.addEventListener("mousedown", off);
    return () => { window.removeEventListener("keydown", k); document.removeEventListener("mousedown", off); };
  }, []);
  if (cap) return <div className="am cap-w" ref={root}><CapturePanel up={up} mode={cap} onBack={() => setCap(null)} onClose={onClose} /></div>;
  return (
    <div className="am" ref={root}>
      <input ref={inp} type="file" multiple hidden onChange={e => { up.add(e.target.files); e.target.value = ""; onClose(); }} />
      <button onClick={() => inp.current.click()}><Ic n="plus" s={14} /><span><b>Upload from computer</b><em>{`PDF, images, CSV, Excel · up to ${MAX_ATTACHMENTS} files, ${MAX_MB} MB each`}</em></span></button>
      <button onClick={() => setCap("ok")}><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M4 15h16v4H4zM6 15V9l4-4h8v10"></path><path d="M8 18h.01"></path></svg><span><b>Scan from Capture</b><em>Scan paper on AVERY-LAPTOP straight into this message</em></span></button>
      <button onClick={() => { up.add(SAMPLES.slice(0, 1)); onClose(); }}><Ic n="copy" s={14} /><span><b>Try a sample rate confirmation</b><em>rate-con-acme-77812.pdf</em></span></button>
      <button onClick={() => { up.add([{ ...SAMPLES[1], name: "BOL-778.pdf", failOnce: true }, SAMPLES[0]]); onClose(); }}><Ic n="replay" s={14} /><span><b>Try a failed upload</b><em>BOL-778.pdf drops partway · retry it</em></span></button>
      <button onClick={() => { up.add([...SAMPLES.slice(1), { ...SAMPLES[0], name: "invoice-sept.pdf", failOnce: true }, ...BAD_SAMPLES]); onClose(); }}><Ic n="alert" s={14} /><span><b>Try a messy batch</b><em>6 files · shows every upload state</em></span></button>
      <div className="am-f"><span className="kbd">⌘U</span> or drop files anywhere</div>
    </div>
  );
}

function DropOverlay({ show, count, hot }) {
  return (
    <div className={"dz" + (show ? " on" : "") + (hot ? " hot" : "")} aria-hidden={!show}>
      <div className="dz-in">
        <div className="dz-stack"><span></span><span></span><span></span></div>
        <b>{hot ? "Release to attach to your message" : `Drop to attach${count ? ` ${count} file${count > 1 ? "s" : ""}` : ""}`}</b>
        <span>{`PDF, images, CSV or Excel · up to ${MAX_ATTACHMENTS} files, ${MAX_MB} MB each`}</span>
      </div>
    </div>
  );
}

function MsgAtts({ atts }) {
  if (!atts || !atts.length) return null;
  return <div className="ma">{atts.map(a => <span key={a.id} className="fp s-sent" data-tip={fmtSize(a.size)}><FileIcon a={a} s={18} /><span className="fp-n">{a.name}</span></span>)}</div>;
}

/* extraction artifact */
const EX_FIELDS = [
  ["Customer", "Acme Manufacturing", 0.99, [8, 15, 44, 4]],
  ["Load / PO", "PO 77812", 0.98, [62, 15, 30, 4]],
  ["Pickup", "Acme DC 4 · Chicago, IL", 0.97, [8, 30, 40, 7]],
  ["Pickup date", "Oct 6, 8:00 AM", 0.95, [52, 30, 30, 4]],
  ["Delivery", "Acme Plant 2 · Columbus, OH", 0.96, [8, 42, 40, 7]],
  ["Delivery date", "Oct 7, 5:30 PM", 0.71, [52, 42, 30, 4]],
  ["Equipment", "53' Dry van", 0.94, [8, 56, 26, 4]],
  ["Weight", "38,400 lb", 0.93, [40, 56, 22, 4]],
  ["Linehaul", "$1,014.60", 0.99, [62, 66, 28, 4]],
  ["Fuel surcharge", "$344.96", 0.98, [62, 72, 28, 4]],
  ["Accessorials", "Detention $65/h after 2h", 0.62, [8, 80, 50, 4]],
  ["Total", "$1,359.56", 0.99, [62, 84, 28, 5]],
];

function ExtractBody({ a }) {
  const [hot, setHot] = React.useState(null);
  const [page, setPage] = React.useState(1);
  const [made, setMade] = React.useState(false);
  const low = EX_FIELDS.filter(f => f[2] < 0.8);
  return (
    <div className="ax-pad ex">
      <div className="ex-h"><span className="ex-type">Rate confirmation</span><span>{a.file} · {a.pages} pages · classified at 98%</span></div>
      <div className="ex-grid">
        <div className="ex-doc">
          <div className="ex-page">
            <div className="ex-ph"><i style={{ width: "34%" }}></i><i style={{ width: "22%", marginLeft: "auto" }}></i></div>
            {Array.from({ length: 22 }, (_, i) => <i key={i} className="ex-ln" style={{ width: [72, 64, 80, 40, 76, 58, 69, 33][i % 8] + "%", marginTop: i % 6 === 0 ? 10 : 4 }}></i>)}
            {page === 1 && EX_FIELDS.map(([l, v, c, [x, y, w, h]], i) => (
              <span key={l} className={"ex-box" + (c < 0.8 ? " low" : "") + (hot === i ? " on" : "")} style={{ left: x + "%", top: y + "%", width: w + "%", height: h + "%" }} onMouseEnter={() => setHot(i)} onMouseLeave={() => setHot(null)}></span>
            ))}
          </div>
          <div className="ex-pg">{Array.from({ length: a.pages }, (_, i) => <button key={i} className={page === i + 1 ? "on" : ""} onClick={() => setPage(i + 1)}>{i + 1}</button>)}</div>
        </div>
        <div className="ex-fields">
          {low.length > 0 && <div className="ex-warn"><Ic n="alert" s={12} w={2} />{low.length} fields need a look</div>}
          {EX_FIELDS.map(([l, v, c], i) => (
            <div key={l} className={"ex-f" + (c < 0.8 ? " low" : "") + (hot === i ? " on" : "")} onMouseEnter={() => { setHot(i); setPage(1); }} onMouseLeave={() => setHot(null)}>
              <span className="ex-l">{l}</span>
              <span className="ex-v">{v}</span>
              <span className="ex-c" title={Math.round(c * 100) + "% confident"}><i style={{ width: c * 100 + "%" }}></i></span>
            </div>
          ))}
        </div>
      </div>
      <div className="ax-acts">
        <button className="ax-btn ghost">Fix fields</button>
        <span style={{ flex: 1 }}></span>
        {made ? <span className="ax-sent"><Ic n="check" s={13} w={2.4} />Shipment drafted · waiting on your approval</span> : <button className="ax-btn ink" onClick={() => setMade(true)}>Create shipment from this</button>}
      </div>
    </div>
  );
}

Object.assign(window, { CapturePanel, useUploads, AttachRow, AttachMenu, DropOverlay, MsgAtts, FileChip, FileIcon, ExtractBody, SAMPLES, BAD_SAMPLES, MAX_ATTACHMENTS, Body_extract: ExtractBody });
