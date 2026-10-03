// Streaming-safe markdown: ATX + setext headings, lists (nested, task), tables, code fences, quotes, hr,
// inline bold/italic/code/strike, inline + reference links, hard line breaks, escapes, TeX math via KaTeX.
const MD_NUM = /^[-+]?[$€£]?\(?[\d,]+(\.\d+)?\)?%?$|^—$|^-$/;
const isSep = l => /^\|?\s*:?-{1,}:?\s*(\|\s*:?-*:?\s*)*\|?\s*$/.test(l) && l.includes("-");
const cells = l => l.trim().replace(/^\|/, "").replace(/\|$/, "").split("|").map(c => c.trim());
const REF_DEF = /^\s{0,3}\[([^\]^][^\]]*)\]:\s*<?(\S+?)>?(?:\s+["'(](.+)["')])?\s*$/;
const BLOCK_START = /^(#{1,6}\s|\s*```|\s*>|\s*\||\s*([-*+]|\d+[.)])\s|\s*\$\$|\s*\\\[)/;

function mdParse(src, refs = {}) {
  const L = src.replace(/\r/g, "").split("\n").filter(l => { const m = l.match(REF_DEF); if (m) { refs[m[1].toLowerCase()] = { url: m[2], title: m[3] }; return false; } return true; });
  const out = [];
  let i = 0;
  while (i < L.length) {
    const l = L[i];
    if (!l.trim()) { i++; continue; }
    let m;
    if ((m = l.match(/^\s*```\s*([\w+-]*)/))) {
      const lang = m[1]; const body = []; i++;
      while (i < L.length && !/^\s*```/.test(L[i])) body.push(L[i++]);
      i++;
      out.push({ t: "code", lang, body: body.join("\n") });
      continue;
    }
    if (/^\s*(\$\$|\\\[)/.test(l)) {
      const close = /^\s*\$\$/.test(l) ? "$$" : "\\]";
      let s = l.trim().slice(2); const body = [];
      if (s.trim().endsWith(close)) { out.push({ t: "math", x: s.trim().slice(0, -2) }); i++; continue; }
      if (s.trim()) body.push(s);
      i++;
      let done = false;
      while (i < L.length) { const t = L[i++]; if (t.trim().endsWith(close)) { body.push(t.trim().slice(0, -2)); done = true; break; } body.push(t); }
      out.push({ t: "math", x: body.join("\n"), open: !done });
      continue;
    }
    if ((m = l.match(/^(#{1,6})\s+(.*?)\s*#*\s*$/))) { out.push({ t: "h", n: m[1].length, x: m[2] }); i++; continue; }
    if (/^\s*([-*_])(\s*\1){2,}\s*$/.test(l)) { out.push({ t: "hr" }); i++; continue; }
    if (/^\s*\|/.test(l) && (i + 1 >= L.length || isSep(L[i + 1]) || isSep(L[i + 1] + "-"))) {
      const head = cells(l); i++;
      const sep = i < L.length ? cells(L[i]) : []; i++;
      const align = head.map((_, k) => { const s = sep[k] || ""; return s.endsWith(":") ? (s.startsWith(":") ? "center" : "right") : null; });
      const rows = [];
      while (i < L.length && /^\s*\|/.test(L[i])) rows.push(cells(L[i++]));
      const num = head.map((_, k) => rows.length > 0 && rows.every(r => !r[k] || MD_NUM.test(r[k].replace(/\*/g, ""))));
      out.push({ t: "table", head, align, rows, num });
      continue;
    }
    if (/^\s*>/.test(l)) {
      const body = [];
      while (i < L.length && /^\s*>/.test(L[i])) body.push(L[i++].replace(/^\s*>\s?/, ""));
      out.push({ t: "quote", kids: mdParse(body.join("\n"), refs) });
      continue;
    }
    if (/^\s*([-*+]|\d+[.)])\s+/.test(l)) {
      const items = [];
      while (i < L.length && (/^\s*([-*+]|\d+[.)])\s+/.test(L[i]) || (/^\s{2,}\S/.test(L[i]) && items.length))) {
        const ln = L[i];
        const ind = ln.match(/^\s*/)[0].length;
        const mm = ln.match(/^\s*([-*+]|\d+[.)])\s+(.*)/);
        if (!mm) { items[items.length - 1].x += "\n" + ln.trim(); i++; continue; }
        let x = mm[2], task = null;
        const tm = x.match(/^\[([ xX])\]\s+(.*)/);
        if (tm) { task = tm[1] !== " "; x = tm[2]; }
        items.push({ ind, ord: /\d/.test(mm[1]), start: parseInt(mm[1]) || 1, x, task });
        i++;
      }
      out.push(mdNest(items));
      continue;
    }
    const para = []; let setext = 0;
    while (i < L.length && L[i].trim() && (!para.length || !BLOCK_START.test(L[i]))) {
      const nx = L[i + 1];
      para.push(L[i++]);
      if (nx != null && /^\s{0,3}=+\s*$/.test(nx)) { setext = 1; i++; break; }
      if (nx != null && /^\s{0,3}-+\s*$/.test(nx)) { setext = 2; i++; break; }
    }
    if (setext) out.push({ t: "h", n: setext, x: para.map(s => s.trim()).join(" ") });
    else out.push({ t: "p", x: para.map(s => s.replace(/^\s+/, "")).join("\n") });
  }
  return out;
}

function mdNest(items) {
  const base = items[0].ind;
  const list = { t: "list", ord: items[0].ord, start: items[0].start, items: [] };
  for (let k = 0; k < items.length; k++) {
    const it = items[k];
    if (it.ind > base + 1 && list.items.length) {
      const kids = [];
      while (k < items.length && items[k].ind > base + 1) kids.push(items[k++]);
      k--;
      list.items[list.items.length - 1].sub = mdNest(kids);
    } else list.items.push({ x: it.x, task: it.task });
  }
  return list;
}

function Tex({ x, display, className }) {
  const html = React.useMemo(() => {
    if (!window.katex) return null;
    try { return window.katex.renderToString(x, { displayMode: !!display, throwOnError: false, output: "html" }); } catch (e) { return null; }
  }, [x, display]);
  if (!html) return <code className={className}>{x}</code>;
  return <span className={(display ? "md-tex-d" : "md-tex") + (className ? " " + className : "")} dangerouslySetInnerHTML={{ __html: html }}></span>;
}

// Inline rules; earliest match wins, ties go to the earlier rule.
const MD_RULES = [
  [/\\([\\`*_{}\[\]()#+\-.!$|~<>])/, (m, c, k) => mdWords(m[1], c, k)],
  [/`([^`]+)`/, (m, c, k) => <code key={k} className={c.w}>{m[1]}</code>],
  [/\$\$([^$]+?)\$\$/, (m, c, k) => <Tex key={k} x={m[1]} className={c.w} />],
  [/\\\((.+?)\\\)/, (m, c, k) => <Tex key={k} x={m[1]} className={c.w} />],
  [/(?<![\w$\\])\$(?=[^\s$])((?:\\\$|[^$\n])*?[^\s$\\])\$(?![\d\w])/, (m, c, k) => <Tex key={k} x={m[1]} className={c.w} />],
  [/(?<!\S)\$([^\s$])\$(?![\d\w])/, (m, c, k) => <Tex key={k} x={m[1]} className={c.w} />],
  [/(\*\*\*|___)(?=\S)([\s\S]+?)(?<=\S)\1/, (m, c, k) => <strong key={k}><em>{mdInline(m[2], c, k)}</em></strong>],
  [/(\*\*|__)(?=\S)([\s\S]+?)(?<=\S)\1/, (m, c, k) => <strong key={k}>{mdInline(m[2], c, k)}</strong>],
  [/~~(?=\S)([\s\S]+?)(?<=\S)~~/, (m, c, k) => <s key={k}>{mdInline(m[1], c, k)}</s>],
  [/\*(?=[^\s*])([\s\S]*?[^\s*])\*/, (m, c, k) => <em key={k}>{mdInline(m[1], c, k)}</em>],
  [/(?<![\w])_(?=[^\s_])([\s\S]*?[^\s_])_(?![\w])/, (m, c, k) => <em key={k}>{mdInline(m[1], c, k)}</em>],
  [/\[([^\]]+)\]\(\s*<?([^)\s>]+)>?(?:\s+["']([^"']*)["'])?\s*\)/, (m, c, k) => mdLink(m[1], m[2], m[3], c, k)],
  [/\[([^\]]+)\]\s?\[([^\]]*)\]/, (m, c, k) => { const r = c.refs && c.refs[(m[2] || m[1]).toLowerCase()]; return r ? mdLink(m[1], r.url, r.title, c, k) : mdWords(m[0], c, k); }],
  [/\[([^\]]+)\](?![(\[:])/, (m, c, k) => { const r = c.refs && c.refs[m[1].toLowerCase()]; return r ? mdLink(m[1], r.url, r.title, c, k) : mdWords(m[0], c, k); }],
  [/<(https?:\/\/[^\s>]+)>/, (m, c, k) => mdLink(m[1], m[1], null, c, k)],
];
function mdLink(text, url, title, c, k) {
  return <a key={k} href={url} title={title || url} target="_blank" rel="noreferrer">{mdInline(text, c, k)}</a>;
}
function mdInline(s, ctx, key = "i") {
  const out = []; let rest = s; let n = 0;
  while (rest) {
    let best = null, fn = null;
    for (const [re, f] of MD_RULES) { const m = rest.match(re); if (m && (!best || m.index < best.index)) { best = m; fn = f; } }
    if (!best) { out.push(mdWords(rest, ctx, key + n++)); break; }
    if (best.index) out.push(mdWords(rest.slice(0, best.index), ctx, key + n++));
    out.push(fn(best, ctx, key + n++));
    rest = rest.slice(best.index + best[0].length);
  }
  return out;
}
// Text run: newlines become hard breaks (trailing "  " or "\" markers stripped).
function mdWords(s, ctx, key) {
  const lines = s.split(/ {2,}\n|\\\n|\n/);
  return (
    <React.Fragment key={key}>
      {lines.map((ln, li) => (
        <React.Fragment key={li}>
          {li > 0 && <br />}
          {ctx.w ? (ln.match(/\S+\s*|\s+/g) || []).map((w, j) => <span key={j} className="w">{w}</span>) : ln}
        </React.Fragment>
      ))}
    </React.Fragment>
  );
}

function MdCode({ b }) {
  const [ok, setOk] = React.useState(false);
  React.useEffect(() => { if (!ok) return; const h = setTimeout(() => setOk(false), 1400); return () => clearTimeout(h); }, [ok]);
  const copy = () => { try { navigator.clipboard.writeText(b.body); } catch (e) {} setOk(true); };
  return (
    <div className="md-code">
      <div className="md-code-h"><span>{b.lang || "text"}</span><button onClick={copy} className={ok ? "ok" : ""}>{ok ? <><Ic n="check" s={12} w={2.2} />Copied</> : <><Ic n="copy" s={12} />Copy</>}</button></div>
      <pre><code>{b.body}</code></pre>
    </div>
  );
}

function MdList({ b, ctx }) {
  const Tag = b.ord ? "ol" : "ul";
  const task = b.items.some(it => it.task != null);
  return (
    <Tag className={task ? "md-task" : undefined} start={b.ord && b.start !== 1 ? b.start : undefined}>
      {b.items.map((it, k) => (
        <li key={k} className={it.task ? "done" : undefined}>
          {it.task != null && <span className="md-cb">{it.task && <Ic n="check" s={10} w={2.6} />}</span>}
          <span>{mdInline(it.x, ctx, "l" + k)}</span>
          {it.sub && <MdList b={it.sub} ctx={ctx} />}
        </li>
      ))}
    </Tag>
  );
}

function MdTable({ b, ctx }) {
  const al = k => b.align[k] || (b.num[k] ? "right" : undefined);
  const c = { refs: ctx.refs };
  return (
    <div className="md-tw"><table className="md-t">
      <thead><tr>{b.head.map((h, k) => <th key={k} style={{ textAlign: al(k) }}>{mdInline(h, c, "h" + k)}</th>)}</tr></thead>
      <tbody>{b.rows.map((r, j) => <tr key={j}>{b.head.map((_, k) => <td key={k} className={b.num[k] ? "n" : undefined} style={{ textAlign: al(k) }}>{r[k] != null ? mdInline(r[k], c, "c" + k) : ""}</td>)}</tr>)}</tbody>
    </table></div>
  );
}

function MdBlocks({ blocks, ctx }) {
  return blocks.map((b, k) => {
    if (b.t === "h") { const H = "h" + Math.min(b.n + 2, 6); return <H key={k} className={"md-h md-h" + b.n}>{mdInline(b.x, ctx, "h")}</H>; }
    if (b.t === "p") return <p key={k}>{mdInline(b.x, ctx, "p")}</p>;
    if (b.t === "hr") return <hr key={k} />;
    if (b.t === "code") return <MdCode key={k} b={b} />;
    if (b.t === "math") return <div key={k} className="md-math">{b.open ? <pre className="md-math-raw">{b.x}</pre> : <Tex x={b.x} display />}</div>;
    if (b.t === "table") return <MdTable key={k} b={b} ctx={ctx} />;
    if (b.t === "quote") return <blockquote key={k}><MdBlocks blocks={b.kids} ctx={ctx} /></blockquote>;
    if (b.t === "list") return <MdList key={k} b={b} ctx={ctx} />;
    return null;
  });
}

function Markdown({ src, streaming }) {
  const [blocks, refs] = React.useMemo(() => { const r = {}; return [mdParse(src, r), r]; }, [src]);
  return <div className="md"><MdBlocks blocks={blocks} ctx={{ w: streaming ? "w" : undefined, refs }} /></div>;
}

Object.assign(window, { mdParse, Markdown, Tex });
