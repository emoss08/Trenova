/* One context menu for the whole app. openCtx(event, { title, items }) — items: { l, ic, k, on, sub, d, sep, hint, dis } */
function CtxHost() {
  const [st, setSt] = React.useState(null);
  const [sub, setSub] = React.useState(null);
  const ref = React.useRef();
  React.useEffect(() => {
    window.openCtx = (e, spec) => { e.preventDefault(); e.stopPropagation(); const root = document.querySelector(".tv").getBoundingClientRect(); setSub(null); setSt({ x: e.clientX - root.left, y: e.clientY - root.top, w: root.width, h: root.height, ...spec }); };
    const close = e => { if (ref.current && ref.current.contains(e.target)) return; setSt(null); };
    const key = e => { if (e.key === "Escape") setSt(null); };
    document.addEventListener("mousedown", close); window.addEventListener("keydown", key, true); window.addEventListener("blur", () => setSt(null));
    const sc = () => setSt(null); document.addEventListener("scroll", sc, true);
    return () => { document.removeEventListener("mousedown", close); window.removeEventListener("keydown", key, true); document.removeEventListener("scroll", sc, true); };
  }, []);
  React.useLayoutEffect(() => {
    if (!st || !ref.current) return; const el = ref.current, r = el.getBoundingClientRect(), root = document.querySelector(".tv").getBoundingClientRect();
    let x = st.x, y = st.y; if (x + r.width > st.w - 8) x = st.w - r.width - 8; if (y + r.height > st.h - 8) y = Math.max(8, st.h - r.height - 8);
    el.style.left = x + "px"; el.style.top = y + "px"; el.dataset.flip = x + r.width + 220 > st.w ? "1" : "";
  }, [st]);
  if (!st) return null;
  const run = it => { if (it.dis || it.sub) return; setSt(null); it.on && it.on(); };
  const Item = ({ it, i, inSub }) => it.sep ? <span className="cx-sep"></span> : it.h ? <div className="cx-h">{it.h}</div> : (
    <div className={"cx-i" + (it.d ? " d" : "") + (it.dis ? " dis" : "") + (sub === i && !inSub ? " on" : "")} onMouseEnter={() => !inSub && setSub(it.sub ? i : null)} onClick={() => run(it)}>
      <span className="cx-ic">{it.icon || (it.ic ? <Ic n={it.ic} s={13}></Ic> : null)}</span>
      <span className="cx-l">{it.l}{it.hint && <em>{it.hint}</em>}</span>
      {it.k && <span className="kbd">{it.k}</span>}
      {it.sub && <Ic n="chevR" s={11}></Ic>}
      {it.sub && sub === i && !inSub && <div className="cx sub" ref={el => { if (!el || el.dataset.fit) return; el.dataset.fit = 1; const root = document.querySelector(".tv").getBoundingClientRect(), r = el.getBoundingClientRect(), mh = Math.min(340, root.height - 16); el.style.maxHeight = mh + "px"; const r2 = el.getBoundingClientRect(), over = r2.bottom - (root.bottom - 8); if (over > 0) el.style.top = (-6 - over) + "px"; }} onMouseDown={e => e.stopPropagation()}>{it.sub.map((s, j) => <Item key={j} it={s} i={j} inSub></Item>)}</div>}
    </div>);
  return ReactDOM.createPortal(
    <div className="cx" ref={ref} onContextMenu={e => e.preventDefault()}>
      {st.title && <div className="cx-t"><b>{st.title}</b>{st.sub && <span>{st.sub}</span>}</div>}
      {st.items.filter(Boolean).map((it, i) => <Item key={i} it={it} i={i}></Item>)}
    </div>, document.querySelector(".tv"));
}
const ctx = (e, spec) => window.openCtx && window.openCtx(e, spec);
Object.assign(window, { CtxHost, ctx });
