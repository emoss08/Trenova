/* Reusable visualization + UI components */

// Sparkline path generator
function sparkPath(data, w = 80, h = 22, pad = 2) {
  if (!data?.length) return { line: "", area: "" };
  const min = Math.min(...data), max = Math.max(...data);
  const range = max - min || 1;
  const step = (w - pad*2) / (data.length - 1);
  const pts = data.map((v, i) => [pad + i*step, h - pad - ((v - min)/range)*(h - pad*2)]);
  const line = pts.map((p,i) => (i ? "L" : "M") + p[0].toFixed(1) + "," + p[1].toFixed(1)).join(" ");
  const area = line + ` L ${pts[pts.length-1][0]},${h} L ${pts[0][0]},${h} Z`;
  return { line, area };
}

const Sparkline = ({ data, color = "var(--brand)", w = 84, h = 22, fill = true }) => {
  const { line, area } = sparkPath(data, w, h);
  const id = React.useId();
  return (
    <svg className="spark" width={w} height={h} viewBox={`0 0 ${w} ${h}`} preserveAspectRatio="none">
      {fill && (
        <>
          <defs>
            <linearGradient id={id} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity="0.30"/>
              <stop offset="100%" stopColor={color} stopOpacity="0"/>
            </linearGradient>
          </defs>
          <path d={area} fill={`url(#${id})`}/>
        </>
      )}
      <path d={line} stroke={color} />
    </svg>
  );
};

// Mini ring/donut
const Ring = ({ value, size = 36, stroke = 4, color = "var(--brand)", trackColor = "color-mix(in oklch, currentColor 12%, transparent)" }) => {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const offset = c - (Math.max(0, Math.min(100, value)) / 100) * c;
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle cx={size/2} cy={size/2} r={r} stroke={trackColor} strokeWidth={stroke} fill="none"/>
      <circle cx={size/2} cy={size/2} r={r} stroke={color} strokeWidth={stroke} fill="none"
              strokeDasharray={c} strokeDashoffset={offset} strokeLinecap="round"
              transform={`rotate(-90 ${size/2} ${size/2})`} />
    </svg>
  );
};

// Tiny horizontal bar
const Bar = ({ value, max = 100, color = "var(--brand)", h = 4 }) => (
  <div style={{ height: h, background: "color-mix(in oklch, var(--fg) 8%, transparent)", borderRadius: 999, overflow: "hidden" }}>
    <div style={{ height: "100%", width: `${(value/max)*100}%`, background: color, borderRadius: 999, transition: "width 200ms ease" }}/>
  </div>
);

// Status pill mapping
const STATUS_STYLE = {
  "In Transit": { cls: "pill-soft-brand",   icon: "•" },
  "At Risk":    { cls: "pill-soft-danger",  icon: "▲" },
  "Detention":  { cls: "pill-soft-warning", icon: "◷" },
  "Loading":    { cls: "pill-soft-muted",   icon: "↑" },
  "Delivered":  { cls: "pill-soft-success", icon: "✓" },
  "Unassigned": { cls: "pill-soft-muted",   icon: "○" },
};
const StatusPill = ({ status }) => {
  const s = STATUS_STYLE[status] || STATUS_STYLE["Unassigned"];
  return <span className={`pill ${s.cls}`}><span style={{fontSize:8}}>{s.icon}</span>{status}</span>;
};

/* ---------- KPI components ---------- */

const toneColor = (t) => ({
  success: "var(--success)", danger: "var(--danger)", warning: "var(--warning)",
  brand: "var(--brand)", info: "var(--info)", muted: "var(--fg-muted)"
}[t] || "var(--fg-muted)");

const KpiHeader = ({ icon, label, right }) => (
  <div style={{display:"flex", alignItems:"center", justifyContent:"space-between", minHeight: 14}}>
    <div className="label" style={{display:"flex", alignItems:"center", gap:6}}>{icon}{label}</div>
    {right}
  </div>
);

const Delta = ({ delta, deltaLabel, deltaTone }) => {
  if (delta === undefined || delta === null) return null;
  const positive = delta >= 0;
  const color = deltaTone ? toneColor(deltaTone) : (positive ? "var(--success)" : "var(--danger)");
  return (
    <span className="mono tabular" style={{ fontSize: 10.5, color, display:"inline-flex", alignItems:"center", gap:2, padding:"1px 5px", borderRadius:3, background:`color-mix(in oklch, ${color} 12%, transparent)` }}>
      {positive ? "▲" : "▼"}{Math.abs(delta)}{deltaLabel || ""}
    </span>
  );
};

// 1. Hero: big number, optional segmented breakdown bar, sparkline footer
const KpiHero = ({ label, value, unit, delta, deltaLabel, deltaTone, sub, sparkData, sparkColor, icon, breakdown, span = 3 }) => (
  <div className="card" style={{ gridColumn:`span ${span}`, padding: 12, display: "flex", flexDirection: "column", gap: 10, height:"var(--kpi-h)" }}>
    <KpiHeader icon={icon} label={label} right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone}/>}/>
    <div style={{ display: "flex", alignItems: "baseline", gap: 4 }}>
      <div className="mono tabular" style={{ fontSize: 28, fontWeight: 600, letterSpacing: "-0.02em", lineHeight: 1 }}>{value}</div>
      {unit && <div className="mono" style={{ fontSize: 12, color: "var(--fg-muted)" }}>{unit}</div>}
    </div>
    {breakdown && <SegmentedBar segments={breakdown}/>}
    <div style={{ marginTop:"auto", display:"flex", alignItems:"flex-end", justifyContent:"space-between", gap: 8 }}>
      <div style={{ fontSize: 10.5, color: "var(--fg-subtle)", lineHeight: 1.35 }}>{sub}</div>
      {sparkData && !breakdown && <Sparkline data={sparkData} color={sparkColor || "var(--brand)"} w={88} h={24}/>}
    </div>
  </div>
);

// 2. Ring: value + ring on left, delta + sub stacked
const KpiRing = ({ label, value, unit, delta, deltaLabel, deltaTone, sub, ringValue, ringMax = 100, target, icon, span = 2 }) => {
  const pct = Math.min(100, (ringValue / ringMax) * 100);
  const onTarget = target ? ringValue >= target : true;
  const ringColor = onTarget ? "var(--success)" : "var(--warning)";
  return (
    <div className="card" style={{ gridColumn:`span ${span}`, padding: 12, display:"flex", flexDirection:"column", gap: 8, height: "var(--kpi-h)" }}>
      <KpiHeader icon={icon} label={label} right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone}/>}/>
      <div style={{ display:"flex", alignItems:"center", gap: 10, marginTop: 2 }}>
        <Ring value={pct} color={ringColor} size={42} stroke={4}/>
        <div style={{ display:"flex", flexDirection:"column", gap: 2, minWidth: 0 }}>
          <div style={{ display:"flex", alignItems:"baseline", gap: 3 }}>
            <div className="mono tabular" style={{ fontSize: 22, fontWeight: 600, letterSpacing: "-0.02em", lineHeight: 1 }}>{value}</div>
            {unit && <div className="mono" style={{ fontSize: 11, color: "var(--fg-muted)" }}>{unit}</div>}
          </div>
          {target && <div className="mono" style={{ fontSize: 9.5, color:"var(--fg-subtle)", letterSpacing:".04em", textTransform:"uppercase" }}>Target {target}{unit}</div>}
        </div>
      </div>
      <div style={{ marginTop:"auto", fontSize: 10.5, color:"var(--fg-subtle)", lineHeight: 1.35 }}>{sub}</div>
    </div>
  );
};

// 3. Goal bar: actual vs target as horizontal range
const KpiGoalBar = ({ label, value, unit, delta, deltaLabel, deltaTone, sub, actual, target, max, icon, span = 2 }) => {
  const actualPct = Math.min(100, (actual / max) * 100);
  const targetPct = Math.min(100, (target / max) * 100);
  // For metrics where lower-is-better (empty mile), color is good when actual <= target
  const onGoal = actual <= target;
  const fillColor = onGoal ? "var(--success)" : "var(--warning)";
  return (
    <div className="card" style={{ gridColumn:`span ${span}`, padding: 12, display:"flex", flexDirection:"column", gap: 8, height:"var(--kpi-h)" }}>
      <KpiHeader icon={icon} label={label} right={<Delta delta={delta} deltaLabel={deltaLabel} deltaTone={deltaTone}/>}/>
      <div style={{ display:"flex", alignItems:"baseline", gap: 4 }}>
        <div className="mono tabular" style={{ fontSize: 22, fontWeight: 600, letterSpacing: "-0.02em", lineHeight: 1 }}>{value}</div>
        {unit && <div className="mono" style={{ fontSize: 11, color: "var(--fg-muted)" }}>{unit}</div>}
      </div>
      <div style={{ position:"relative", height: 6, background:"var(--bg-2)", borderRadius: 3, overflow:"visible", marginTop: 2 }}>
        <div style={{ position:"absolute", left:0, top:0, bottom:0, width:`${actualPct}%`, background: fillColor, borderRadius: 3 }}/>
        <div title={`Target ${target}${unit||""}`} style={{ position:"absolute", left:`calc(${targetPct}% - 1px)`, top:-2, bottom:-2, width: 2, background:"var(--fg)", opacity: .55, borderRadius: 1 }}/>
      </div>
      <div style={{ marginTop:"auto", fontSize: 10.5, color:"var(--fg-subtle)", lineHeight: 1.35 }}>{sub}</div>
    </div>
  );
};

// 4. Stat: compact number-forward card, no chart
const KpiStat = ({ label, value, delta, deltaLabel, sub, tone = "brand", icon, span = 2 }) => {
  const dot = toneColor(tone);
  return (
    <div className="card" style={{ gridColumn:`span ${span}`, padding: 12, display:"flex", flexDirection:"column", gap: 8, height: "var(--kpi-h-sm)" }}>
      <KpiHeader
        icon={<span style={{display:"inline-flex",alignItems:"center",gap:6}}><span style={{width:6,height:6,borderRadius:99,background:dot,display:"inline-block"}}/>{icon}</span>}
        label={label}
        right={<Delta delta={delta} deltaLabel={deltaLabel}/>}
      />
      <div className="mono tabular" style={{ fontSize: 26, fontWeight: 600, letterSpacing: "-0.02em", lineHeight: 1 }}>{value}</div>
      <div style={{ marginTop:"auto", fontSize: 10.5, color:"var(--fg-subtle)", lineHeight: 1.35 }}>{sub}</div>
    </div>
  );
};

// 5. Watchlist: stacked mini-list of items (drivers, shipments) — useful when individual rows matter more than a single number
const KpiWatchlist = ({ label, items, icon, span = 3 }) => (
  <div className="card" style={{ gridColumn:`span ${span}`, padding:"10px 12px 8px", display:"flex", flexDirection:"column", gap: 6, height: "var(--kpi-h-sm)" }}>
    <KpiHeader icon={icon} label={label} right={<span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>{items.length}</span>}/>
    <div style={{display:"flex", flexDirection:"column", gap: 3, marginTop: 2}}>
      {items.map((it, i) => (
        <div key={i} style={{display:"flex", alignItems:"center", justifyContent:"space-between", gap:8, padding:"3px 6px", borderRadius:3, background: i===0 ? "color-mix(in oklch, var(--fg) 4%, transparent)" : "transparent"}}>
          <span style={{display:"flex", alignItems:"center", gap:6, minWidth:0, overflow:"hidden"}}>
            <span style={{width:5,height:5,borderRadius:99,background:toneColor(it.tone),flex:"none"}}/>
            <span className="mono" style={{fontSize:11, color:"var(--fg)", overflow:"hidden", textOverflow:"ellipsis", whiteSpace:"nowrap"}}>{it.who}</span>
          </span>
          <span className="mono tabular" style={{fontSize:10.5, color: toneColor(it.tone), flex:"none"}}>{it.meta}</span>
        </div>
      ))}
    </div>
  </div>
);

// Segmented bar for Hero breakdown
const SegmentedBar = ({ segments }) => {
  const total = segments.reduce((a,b) => a + b.value, 0) || 1;
  return (
    <div style={{display:"flex", flexDirection:"column", gap: 4}}>
      <div style={{display:"flex", height: 6, borderRadius: 3, overflow:"hidden", background:"var(--bg-2)"}}>
        {segments.map((s,i) => (
          <div key={i} title={`${s.label}: ${s.value}`} style={{flex: s.value/total, background: s.color}}/>
        ))}
      </div>
      <div style={{display:"flex", flexWrap:"wrap", gap:"2px 10px"}}>
        {segments.map((s,i) => (
          <span key={i} style={{display:"inline-flex", alignItems:"center", gap:4, fontSize:9.5, color:"var(--fg-subtle)", letterSpacing:".02em"}}>
            <span style={{width:5,height:5,borderRadius:1,background:s.color}}/>
            {s.label} <span className="mono tabular" style={{color:"var(--fg-muted)"}}>{s.value}</span>
          </span>
        ))}
      </div>
    </div>
  );
};

// Sidebar nav item
const NavItem = ({ icon, label, active, count }) => (
  <div style={{
    display:"flex", alignItems:"center", gap:10, padding:"6px 10px", borderRadius:5,
    background: active ? "color-mix(in oklch, var(--fg) 8%, transparent)" : "transparent",
    color: active ? "var(--fg)" : "var(--fg-muted)",
    fontWeight: active ? 600 : 500, fontSize: 12, cursor: "pointer"
  }}>
    {icon}<span style={{flex:1}}>{label}</span>
    {count !== undefined && <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>{count}</span>}
  </div>
);

// Lane heatmap cell
const HeatCell = ({ value, max }) => {
  const t = max ? value / max : 0;
  const bg = `color-mix(in oklch, var(--brand) ${Math.round(t*100)}%, transparent)`;
  const fg = t > 0.55 ? "white" : "var(--fg)";
  return (
    <div className="heat-cell mono tabular" style={{
      height: 26, display:"flex", alignItems:"center", justifyContent:"center",
      background: bg, borderRadius: 3, fontSize: 11, color: value ? fg : "var(--fg-subtle)",
      border: "1px solid var(--border-2)"
    }}>{value || "·"}</div>
  );
};

Object.assign(window, { Sparkline, Ring, Bar, StatusPill, KpiHero, KpiRing, KpiGoalBar, KpiStat, KpiWatchlist, SegmentedBar, NavItem, HeatCell, sparkPath });
