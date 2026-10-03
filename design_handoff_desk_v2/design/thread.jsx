function tokenize(text) {
  const out = [];
  text.forEach((para, p) => {
    if (typeof para === "string") { (para.match(/\S+\s*|\s+/g) || []).forEach(w => out.push({ p, w, md: true })); return; }
    para.forEach(seg => {
    if (seg && seg.a) { out.push({ p, art: seg.a, w: "" }); return; }
    const s = typeof seg === "string" ? { t: seg } : seg.b ? { t: seg.b, b: true } : seg;
    const words = s.t.match(/\S+\s*|\s+/g) || [];
    words.forEach((w, i) => out.push({ p, w, r: s.r, b: s.b, end: s.r && i === words.length - 1 }));
  });
  });
  return out;
}

function Prose({ def, shown, mode, streaming, onRef, hot, setHot, itemId, onArtOpen, activeArt }) {
  const toks = React.useMemo(() => tokenize(def.text), [def]);
  const vis = toks.slice(0, shown == null ? toks.length : shown);
  const paras = [];
  vis.forEach((t, i) => { (paras[t.p] = paras[t.p] || []).push([t, i]); });
  const cls = streaming ? "w" : undefined;
  return (
    <div className="prose">
      {paras.map((ws, p) => ws[0][0].md ? <Markdown key={p} src={ws.map(([t]) => t.w).join("")} streaming={streaming} /> : (
        <p key={p}>
          {ws.map(([t, i]) => {
            if (t.art) return <button key={i} className={"ibadge" + (cls ? " w" : "") + (activeArt === t.art ? " on" : "")} onClick={() => onArtOpen(t.art)}><Ic n="table" s={12} />{artMeta(t.art).title}</button>;
            const fnHere = t.r && t.end && mode === "ledger";
            const word = fnHere ? t.w.replace(/\s+$/, "") : t.w;
            const trail = fnHere ? t.w.slice(word.length) : "";
            const node = t.b ? <strong>{word}</strong> : word;
            if (t.r && mode === "narrated") return <span key={i} className={"ref" + (cls ? " w" : "")} onClick={() => onRef(t.r)}>{node}</span>;
            const k = itemId + ":" + t.r;
            return (
              <React.Fragment key={i}>
                <span className={cls}>{node}</span>
                {fnHere && <><FnPop n={t.r} step={def.trace[t.r - 1]} open={hot === k} onEnter={() => setHot(k)} onLeave={() => setHot(null)} onOpen={() => onRef(t.r)} />{trail}</>}
              </React.Fragment>
            );
          })}
        </p>
      ))}
    </div>
  );
}

function FnPop({ n, step, open, onEnter, onLeave, onOpen }) {
  const [why, setWhy] = React.useState(false);
  return (
    <span className="fnw" onMouseEnter={onEnter} onMouseLeave={onLeave}>
      <span className={"fn" + (open ? " hot" : "")} onClick={onOpen}>{n}</span>
      {open && step && (
        <span className="fnp" role="tooltip">
          <span className="fnp-in">
            <span className="fnp-h"><span>{String(n).padStart(2, "0")}</span><span>{step.k}</span><span className="t">{step.t}</span></span>
            <span className="fnp-b">{step.done}</span>
            <span className="fnp-q"><span>{step.v}</span><Ic n="chevR" s={10} /><b>{step.r}</b></span>
            {step.art && <button className="abadge sm" onClick={onOpen}><Ic n="table" s={11} />{artMeta(step.art).title}</button>}
            <button className={"fnp-why" + (why ? " on" : "")} onClick={e => { e.stopPropagation(); setWhy(w => !w); }}><Ic n="why" s={12} />{why ? "Hide reasoning" : "Why this step?"}</button>
            {why && <StepWhy step={step} />}
          </span>
        </span>
      )}
    </span>
  );
}

function MsgActions({ pinned, reading, onPin, onRead, chapter }) {
  const [copied, setCopied] = React.useState(false);
  React.useEffect(() => { if (!copied) return; const h = setTimeout(() => setCopied(false), 1400); return () => clearTimeout(h); }, [copied]);
  return (
    <div className={"acts" + (pinned || reading || copied ? " stay" : "")}>
      <button className={"act" + (pinned ? " on" : "")} data-tip={pinned ? "Unpin chapter" : "Pin as chapter"} onClick={onPin}><Ic n="bookmark" s={14} />{pinned && <span className="act-l">Chapter {chapter}</span>}</button>
      <button className={"act" + (copied ? " ok" : "")} data-tip={copied ? "Copied" : "Copy"} onClick={() => setCopied(true)}>{copied ? <Ic n="check" s={14} w={2.2} /> : <Ic n="copy" s={14} />}</button>
      <button className={"act" + (reading ? " on" : "")} data-tip={reading ? "Stop reading" : "Read aloud"} onClick={onRead}>{reading ? <span className="eq">{[0, 1, 2, 3].map(k => <i key={k} style={{ animationDelay: k * 120 + "ms" }}></i>)}</span> : <Ic n="speaker" s={14} />}</button>
    </div>
  );
}

const CONF_COLORS = ["#2f9bff", "#5b5bff", "#b03bff", "#ff3d8b", "#ff7a1a", "#ffc53d", "#22c55e"];
function Confetti({ seed }) {
  const pieces = React.useMemo(() => Array.from({ length: 64 }, (_, k) => {
    const r = (n) => { const x = Math.sin(seed * 97 + k * 13.37 + n * 7.1) * 10000; return x - Math.floor(x); };
    return { x: (r(1) - 0.5) * 900, up: 0, fall: 0, sway: (r(2) - 0.5) * 80, rot: (r(4) - 0.5) * 900, dur: 2800 + r(5) * 1600, delay: r(6) * 700, w: 5 + r(7) * 5, h: r(8) > .5 ? 10 + r(9) * 6 : 5 + r(9) * 3, c: CONF_COLORS[k % 7], round: r(10) > .8 };
  }), [seed]);
  return (
    <div className="confetti" aria-hidden="true">
      {pieces.map((p, k) => <i key={k} style={{ "--x": p.x + "px", "--sway": p.sway + "px", "--rot": p.rot + "deg", width: p.w, height: p.h, background: p.c, borderRadius: p.round ? "50%" : 2, animationDuration: p.dur + "ms", animationDelay: p.delay + "ms" }}></i>)}
    </div>
  );
}

function ArtBadges({ arts, onOpen, active }) {
  if (!arts.length) return null;
  return <div className="abadges">{arts.map(a => <button key={a} className={"abadge" + (active === a ? " on" : "")} onClick={() => onOpen(a)}><ArtKindIcon id={a} s={11} /><span>{artMeta(a).title}</span><em>{artMeta(a).count}</em></button>)}</div>;
}

const norm = (def, it) => (it.phase === "work" && !def.trace[it.step] ? { ...it, phase: "write" } : it);
const poseOf = (def, it) => it.phase === "work" ? def.trace[it.step].pose : it.phase === "write" ? "write" : "done";

function Narr({ def, item: raw, elapsed, onToggle, onArt }) {
  const item = norm(def, raw);
  if (item.phase !== "done") {
    const past = def.trace.slice(0, item.step).slice(-2);
    const label = item.phase === "work" ? def.trace[item.step].live : item.phase === "write" ? "Writing the answer" : "Done";
    return (
      <div className="nar"><div className="nar-live">
        {past.map((s, i) => <div className="nl past" key={s.done}><span className="ck"><Ic n="check" s={12} w={2.2} /></span><span>{s.done}</span></div>)}
        <div className="nl cur" key={"c" + item.step + item.phase}><span className="ck"><span className="cmp-st-d"></span></span><span className="tx">{label}</span><span className="el">{elapsed.toFixed(1)}s</span></div>
      </div></div>
    );
  }
  return (
    <div className="nar">
      <button className={"nsum" + (item.open ? " open" : "")} onClick={onToggle}>
        <span className="chev"><Ic n="chevR" s={12} /></span><span>Worked through {def.trace.length} steps in {def.dur}</span>
      </button>
      <div className={"nexp" + (item.open ? " open" : "")}><div>
        <div className="nlist">
          {def.trace.map((s, i) => <NarrStep key={i} s={s} onArt={onArt} />)}
        </div>
      </div></div>
    </div>
  );
}

function NarrStep({ s, onArt }) {
  const [why, setWhy] = React.useState(false);
  return <>
    <div className={"ni" + (s.art ? " link" : "")} onClick={() => s.art && onArt(s.art)}><span>{s.done} <em>· {s.r}</em><button className={"ni-why" + (why ? " on" : "")} onClick={e => { e.stopPropagation(); setWhy(w => !w); }}>{why ? "Hide" : "Why?"}</button></span><span className="t">{s.t}</span></div>
    {why && <div className="ni-wx"><StepWhy step={s} /></div>}
  </>;
}

function Receipt({ def, item: raw, elapsed, hot, setHot, onArt }) {
  const item = norm(def, raw);
  const live = item.phase !== "done";
  const n = live ? item.step : def.trace.length;
  return (
    <div className="rc">
      <div className="rc-h">{live ? <><span className="cmp-st-d"></span>{item.phase === "work" ? "working" : item.phase === "write" ? "writing" : "done"}<span className="el">{elapsed.toFixed(1)}s</span></> : <>receipt<span className="el">{def.time.replace(" PM", "")}</span></>}</div>
      {def.trace.slice(0, n).map((s, i) => {
        const k = item.id + ":" + (i + 1);
        return (
          <div key={i} className={"rl" + (hot === k ? " hot" : "") + (s.art ? " link" : "")} onMouseEnter={() => setHot(k)} onMouseLeave={() => setHot(null)} onClick={() => s.art && onArt(s.art)}>
            <span className="n">{String(i + 1).padStart(2, "0")}</span><span className="k">{s.k.replace("billing.", "")}</span><span className="v">{s.r.replace(" rows", "")}</span>
          </div>
        );
      })}
      {live && item.phase === "work" && <div className="rl" key={"p" + item.step}><span className="n">{String(item.step + 1).padStart(2, "0")}</span><span className="k">{def.trace[item.step].k.replace("billing.", "")}</span><span className="pend"></span></div>}
      {!live && <div className="rc-f"><span>{n} calls</span><span>{def.dur}</span></div>}
      {!live && def.decision && <div className="rc-x"><Ic n="shield" s={11} />1 change → you</div>}
    </div>
  );
}

function ArtCard({ art, onOpen, delay = 0 }) {
  const a = ARTIFACTS[art];
  const rows = QUEUE.filter(a.filter).slice(0, 7);
  return (
    <button className="ac" style={{ animationDelay: delay + "ms" }} onClick={() => onOpen(art)}>
      <div className="ac-pv">
        <i className="hd"><b style={{ width: 22 }}></b><b style={{ width: 40 }}></b><b style={{ width: 24 }}></b></i>
        {rows.map((r, i) => <i key={i}><b style={{ width: 22 }}></b><b style={{ width: 30 + ((i * 17) % 28) }}></b><b className={art === "gaps" ? "hi" : ""} style={{ width: 18 + ((i * 11) % 12) }}></b></i>)}
      </div>
      <div className="ac-t"><Ic n="table" s={12} />{a.title}</div>
      <div className="ac-s">{QUEUE.filter(a.filter).length} rows · <span className="mono">{a.sub.split("from ")[1]}</span></div>
    </button>
  );
}

Object.assign(window, { Confetti, MsgActions, tokenize, Prose, Narr, Receipt, ArtCard, ArtBadges, norm });
