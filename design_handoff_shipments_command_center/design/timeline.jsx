/* Timeline (Gantt) view for the Shipments Command Center.
   Driver-row swimlanes across a 24h time axis, with shipment blocks,
   stop markers, dwell, HOS exhaustion, and a live "now" line. */

function TimelineView({ shipments, drivers, expandedId, setExpandedId, highlightId, onHover }) {
  const [zoom, setZoom] = React.useState("24h"); // 12h | 24h | 48h
  const [tick, setTick] = React.useState(0);
  React.useEffect(() => { const id = setInterval(() => setTick(t => t + 1), 30000); return () => clearInterval(id); }, []);

  // Scenario anchor — the mock data's "now" lives on Apr 22 ~13:00 of the current year
  // so loads built from those ETAs sit naturally inside the visible window.
  const now = React.useMemo(() => {
    const d = new Date();
    d.setMonth(3, 22); // April (0-indexed) 22
    d.setHours(13, 0, 0, 0);
    return d;
  }, []);
  const span = zoom === "12h" ? 12 : zoom === "48h" ? 48 : 24;
  const start = new Date(now.getTime() - span * 0.5 * 3600 * 1000);
  const end   = new Date(now.getTime() + span * 0.5 * 3600 * 1000);

  // Each driver gets a swimlane. Unassigned shipments get a synthetic "Unassigned" lane.
  const driverShipments = drivers.map(d => {
    const sh = shipments.find(s => s.driver === d.name) || shipments.find(s => s.id === d.load);
    return { driver: d, shipments: sh ? [sh] : [] };
  });
  const unassigned = shipments.filter(s => s.driver === "—");
  const lanes = [
    ...driverShipments,
    ...(unassigned.length ? [{ driver: { id: "UN", name: "Unassigned", tractor: "—", hosLeft: "—", status: "unassigned" }, shipments: unassigned }] : []),
  ];

  return (
    <div className="timeline" style={{display:"flex", flexDirection:"column"}}>
      <TimelineToolbar zoom={zoom} setZoom={setZoom} now={now}/>
      <TimelineGrid
        lanes={lanes}
        start={start} end={end} now={now}
        expandedId={expandedId} setExpandedId={setExpandedId}
        highlightId={highlightId} onHover={onHover}
      />
      <TimelineLegend/>
    </div>
  );
}

function TimelineToolbar({ zoom, setZoom, now }) {
  const fmt = (d) => d.toLocaleTimeString("en-US", { hour: "2-digit", minute:"2-digit", hour12:false });
  return (
    <div style={{padding:"6px 10px", borderBottom:"1px solid var(--border)", display:"flex", gap:8, alignItems:"center"}}>
      <div className="label" style={{fontSize:9.5}}>Window</div>
      <div style={{display:"flex", gap:2, background:"var(--card-2)", border:"1px solid var(--border)", borderRadius:4, padding:2}}>
        {["12h","24h","48h"].map(z => {
          const on = zoom === z;
          return (
            <button key={z} onClick={() => setZoom(z)} style={{
              border:"none", background: on ? "var(--card)" : "transparent",
              color: on ? "var(--fg)" : "var(--fg-muted)", fontWeight: on ? 600 : 500,
              padding:"2px 8px", borderRadius:3, fontSize:11, cursor:"pointer",
              boxShadow: on ? "0 1px 0 var(--border)" : "none",
            }}>{z}</button>
          );
        })}
      </div>
      <div style={{width:1, height:18, background:"var(--border)", margin:"0 2px"}}/>
      <button className="btn" data-tip="Jump to now"><IcPin size={11}/>Now · {fmt(now)}</button>
      <div style={{flex:1}}/>
      <span className="mono" style={{fontSize:10.5, color:"var(--fg-muted)"}}>
        Drag blocks to reschedule · Click for details
      </span>
    </div>
  );
}

function TimelineGrid({ lanes, start, end, now, expandedId, setExpandedId, highlightId, onHover }) {
  const LANE_LABEL_W = 180;
  const ROW_H = 54;
  const totalMs = end - start;
  const pct = (d) => ((d - start) / totalMs) * 100;
  const nowPct = pct(now);

  // Hour ticks
  const hours = [];
  let cur = new Date(start);
  cur.setMinutes(0, 0, 0);
  while (cur <= end) { hours.push(new Date(cur)); cur = new Date(cur.getTime() + 3600 * 1000); }

  return (
    <div style={{position:"relative", overflowX:"auto"}}>
      <div style={{minWidth: 980, position:"relative"}}>
        {/* Header row: hour ticks */}
        <div style={{display:"grid", gridTemplateColumns:`${LANE_LABEL_W}px 1fr`, borderBottom:"1px solid var(--border)", background:"var(--card-2)", position:"sticky", top:0, zIndex:2}}>
          <div className="label" style={{padding:"6px 10px", borderRight:"1px solid var(--border)", fontSize:9.5}}>Driver / Truck</div>
          <div style={{position:"relative", height:28}}>
            {hours.map((h, i) => {
              const left = pct(h);
              const isMidnight = h.getHours() === 0;
              const isMajor = h.getHours() % 6 === 0;
              return (
                <div key={i} style={{
                  position:"absolute", left:`${left}%`, top:0, bottom:0,
                  borderLeft: isMidnight ? "1px solid var(--border)" : isMajor ? "1px dashed var(--border-2)" : "none",
                  display: isMajor ? "flex" : "none", alignItems:"center", paddingLeft:4,
                }}>
                  <span className="mono" style={{fontSize:9.5, color: isMidnight ? "var(--fg-muted)" : "var(--fg-subtle)", whiteSpace:"nowrap"}}>
                    {String(h.getHours()).padStart(2,"0")}:00
                    {isMidnight && <span style={{marginLeft:4, fontWeight:600, color:"var(--fg)"}}>{h.toLocaleDateString("en-US",{month:"short", day:"numeric"})}</span>}
                  </span>
                </div>
              );
            })}
            {/* NOW marker in header */}
            <div style={{position:"absolute", left:`${nowPct}%`, top:0, bottom:0, width:0, zIndex:3}}>
              <div className="mono" style={{position:"absolute", top:4, left:-22, background:"var(--brand)", color:"white", fontSize:9, fontWeight:600, padding:"1px 6px", borderRadius:3, letterSpacing:"0.05em", whiteSpace:"nowrap"}}>NOW</div>
            </div>
          </div>
        </div>

        {/* Lanes */}
        {lanes.map((lane, idx) => (
          <TimelineLane
            key={lane.driver.id} lane={lane} idx={idx}
            start={start} end={end} pct={pct} ROW_H={ROW_H} LANE_LABEL_W={LANE_LABEL_W}
            now={now} hours={hours} nowPct={nowPct}
            expandedId={expandedId} setExpandedId={setExpandedId}
            highlightId={highlightId} onHover={onHover}
          />
        ))}
      </div>
    </div>
  );
}

function TimelineLane({ lane, idx, start, end, pct, ROW_H, LANE_LABEL_W, now, expandedId, setExpandedId, highlightId, onHover, hours, nowPct }) {
  const { driver, shipments } = lane;
  const isUnassigned = driver.id === "UN";

  // Build blocks for each shipment in this lane.
  // We synthesize a window from a "pickup" time → "delivery" time using progress + miles + eta.
  const blocks = shipments.map(s => buildBlocks(s, start, end, now)).flat();

  return (
    <div style={{display:"grid", gridTemplateColumns:`${LANE_LABEL_W}px 1fr`, borderBottom:"1px solid var(--border-2)", background: idx % 2 ? "color-mix(in oklch, var(--fg) 1.5%, transparent)" : "transparent"}}>
      {/* Lane label */}
      <div style={{padding:"6px 10px", borderRight:"1px solid var(--border)", display:"flex", alignItems:"center", gap:8, height:ROW_H}}>
        {isUnassigned ? (
          <>
            <div style={{width:24, height:24, borderRadius:5, background:"var(--warning-soft)", color:"var(--warning)", display:"flex", alignItems:"center", justifyContent:"center"}}>
              <IcAlert size={12}/>
            </div>
            <div style={{flex:1, lineHeight:1.2}}>
              <div style={{fontSize:11.5, fontWeight:600}}>Unassigned</div>
              <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{shipments.length} load{shipments.length === 1 ? "" : "s"} · drag to assign</div>
            </div>
          </>
        ) : (
          <>
            <div style={{width:24, height:24, borderRadius:"50%", background:`linear-gradient(135deg, oklch(0.7 0.15 ${(idx*48)%360}), oklch(0.55 0.18 ${(idx*48+60)%360}))`, color:"white", display:"flex", alignItems:"center", justifyContent:"center", fontSize:9, fontWeight:600}}>
              {driver.name.split(" ").map(p => p[0]).join("").slice(0,2)}
            </div>
            <div style={{flex:1, lineHeight:1.2, minWidth:0}}>
              <div style={{fontSize:11.5, fontWeight:500, whiteSpace:"nowrap", overflow:"hidden", textOverflow:"ellipsis"}}>{driver.name}</div>
              <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", display:"flex", gap:6}}>
                <span>{driver.tractor}</span>
                <span>·</span>
                <HosBadge left={driver.hosLeft}/>
              </div>
            </div>
            <DriverStatusDot status={driver.status}/>
          </>
        )}
      </div>

      {/* Time track */}
      <div style={{position:"relative", height:ROW_H, overflow:"hidden"}}
           onMouseLeave={() => onHover(null)}>
        {/* Hour grid lines */}
        {hours.map((h, i) => {
          const isMidnight = h.getHours() === 0;
          const isMajor = h.getHours() % 6 === 0;
          if (!isMajor) return null;
          return (
            <div key={i} style={{
              position:"absolute", top:0, bottom:0, left:`${pct(h)}%`,
              borderLeft: isMidnight ? "1px solid var(--border)" : "1px dashed var(--border-2)"
            }}/>
          );
        })}

        {/* Shift bands — paint sleep/break time as soft stripes for context */}
        <ShiftBands hours={hours} pct={pct}/>

        {/* HOS exhaustion warning band: if hosLeft is short, show a red-tinted region from now to now+hosLeft */}
        {!isUnassigned && driver.hosLeft && driver.hosLeft !== "—" && (
          <HosExhaustionBand driver={driver} now={now} pct={pct} end={end}/>
        )}

        {/* Shipment blocks */}
        {blocks.map((b, i) => (
          <ShipmentBlock
            key={s_blockKey(b, i)} block={b} pct={pct}
            highlighted={highlightId === b.shipment.id}
            expanded={expandedId === b.shipment.id}
            onClick={() => setExpandedId(expandedId === b.shipment.id ? null : b.shipment.id)}
            onHover={() => onHover(b.shipment.id)}
          />
        ))}

        {/* Now line — drawn per lane so it lives inside the time track */}
        <div style={{position:"absolute", top:0, bottom:0, left:`${nowPct}%`, width:0, pointerEvents:"none", zIndex:5}}>
          <div style={{position:"absolute", top:0, bottom:0, left:-1, width:2, background:"var(--brand)"}}/>
          {idx === 0 && (
            <>
              <div style={{position:"absolute", top:-3, left:-5, width:10, height:10, borderRadius:"50%", background:"var(--brand)", boxShadow:"0 0 0 2px var(--card)"}} className="pulse"/>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function s_blockKey(b, i) { return `${b.shipment.id}-${b.kind}-${i}`; }

/* Build a sequence of blocks for a shipment within the visible window.
   pickup window → in-transit (with optional dwell stripe) → delivery window. */
function buildBlocks(s, winStart, winEnd, now) {
  const out = [];
  const eta = parseEta(s.eta);
  if (!eta) return out;

  // Estimate trip duration from miles (avg 50 mph)
  const tripHours = Math.max(1, s.miles / 50);
  const pickupAt  = new Date(eta.getTime() - tripHours * 3600 * 1000);
  const deliveryAt = eta;

  // For Delivered: shift everything earlier so it sits in the past
  if (s.status === "Delivered") {
    out.push({
      kind: "transit", shipment: s,
      from: pickupAt, to: deliveryAt,
      label: `${s.originCode} → ${s.destCode}`,
      tone: "success",
    });
    return out;
  }

  // Pickup window (1 hour bar)
  const pickupEnd = new Date(pickupAt.getTime() + 60 * 60 * 1000);
  out.push({ kind:"pickup", shipment:s, from: pickupAt, to: pickupEnd, label: s.originCode, tone: s.status === "Loading" ? "info-active" : "info" });

  // In-transit
  out.push({ kind:"transit", shipment:s, from: pickupEnd, to: new Date(deliveryAt.getTime() - 60*60*1000), label: `${s.originCode} → ${s.destCode}`, tone: toneForStatus(s.status) });

  // Detention / dwell stripe — if currently in detention, paint a "stuck" segment around now
  if (s.status === "Detention" && now >= pickupEnd && now <= deliveryAt) {
    const dwellStart = new Date(now.getTime() - (s.dwell || 120) * 60 * 1000);
    out.push({ kind:"dwell", shipment:s, from: dwellStart, to: now, label: `Dwell ${Math.round((s.dwell||120)/60*10)/10}h`, tone: "danger" });
  }

  // Delivery window (1 hour bar)
  out.push({ kind:"delivery", shipment:s, from: new Date(deliveryAt.getTime() - 60*60*1000), to: deliveryAt, label: s.destCode, tone: s.etaStatus === "late" ? "danger" : "success" });

  return out;
}

function toneForStatus(status) {
  if (status === "At Risk") return "warning";
  if (status === "Detention") return "danger";
  if (status === "Loading") return "info-active";
  if (status === "Unassigned") return "muted";
  if (status === "Delivered") return "success";
  return "brand";
}

function parseEta(eta) {
  // "Apr 22, 14:30" — anchor to current year
  if (!eta || eta === "—") return null;
  const m = eta.match(/(\w+)\s+(\d+),\s+(\d+):(\d+)/);
  if (!m) return null;
  const months = ["Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"];
  const mo = months.indexOf(m[1]);
  if (mo < 0) return null;
  const d = new Date();
  d.setMonth(mo, parseInt(m[2]));
  d.setHours(parseInt(m[3]), parseInt(m[4]), 0, 0);
  return d;
}

function ShipmentBlock({ block, pct, highlighted, expanded, onClick, onHover }) {
  const rawLeft = pct(block.from);
  const rawRight = pct(block.to);
  if (rawRight < 0 || rawLeft > 100) return null;
  // Clamp to visible window so blocks that extend past edges don't overflow into the label gutter
  const left = Math.max(0, rawLeft);
  const right = Math.min(100, rawRight);
  const width = Math.max(0.4, right - left);
  const clippedStart = rawLeft < 0;
  const clippedEnd = rawRight > 100;

  const t = block.tone;
  const palette = {
    "brand":   { bg: "color-mix(in oklch, var(--brand) 22%, var(--card))", bd: "var(--brand)", fg: "var(--fg)" },
    "success": { bg: "color-mix(in oklch, var(--success) 22%, var(--card))", bd: "var(--success)", fg: "var(--fg)" },
    "warning": { bg: "color-mix(in oklch, var(--warning) 28%, var(--card))", bd: "var(--warning)", fg: "var(--fg)" },
    "danger":  { bg: "color-mix(in oklch, var(--danger) 28%, var(--card))",  bd: "var(--danger)",  fg: "var(--fg)" },
    "info":    { bg: "color-mix(in oklch, var(--info) 18%, var(--card))",   bd: "var(--info)",    fg: "var(--fg)" },
    "info-active": { bg: "color-mix(in oklch, var(--info) 30%, var(--card))",bd: "var(--info)",   fg: "var(--fg)" },
    "muted":   { bg: "color-mix(in oklch, var(--fg) 8%, var(--card))",      bd: "var(--fg-subtle)", fg: "var(--fg-muted)" },
  }[t] || { bg: "var(--card-2)", bd: "var(--border)", fg: "var(--fg)" };

  // Markers (pickup/delivery) render as compact diamond/arrow icons
  if (block.kind === "pickup" || block.kind === "delivery") {
    const isPickup = block.kind === "pickup";
    const markerPos = isPickup ? rawLeft : rawRight;
    if (markerPos < 0 || markerPos > 100) return null;
    return (
      <div
        onClick={onClick} onMouseEnter={onHover}
        data-tip={`${isPickup ? "Pickup" : "Delivery"} · ${block.label} · ${fmtTime(isPickup ? block.from : block.to)}`}
        style={{
          position:"absolute", top:"50%", left:`${markerPos}%`, transform:"translate(-50%, -50%)",
          width:18, height:18, borderRadius: isPickup ? "3px" : "50%",
          background: palette.bg, border:`1.5px solid ${palette.bd}`,
          display:"flex", alignItems:"center", justifyContent:"center",
          cursor:"pointer", zIndex: highlighted ? 4 : 2,
        }}
      >
        <span style={{fontSize:8, fontWeight:700, color: palette.bd, letterSpacing:"-0.03em"}}>
          {isPickup ? "P" : "D"}
        </span>
      </div>
    );
  }

  // Dwell / detention stripe
  if (block.kind === "dwell") {
    return (
      <div
        onClick={onClick} onMouseEnter={onHover}
        data-tip={`${block.label} · ${block.shipment.id}`}
        style={{
          position:"absolute", left:`${left}%`, width:`${width}%`,
          top: 8, bottom: 8, borderRadius:3,
          background: `repeating-linear-gradient(45deg, color-mix(in oklch, var(--danger) 30%, transparent) 0 6px, transparent 6px 12px)`,
          border:`1px dashed var(--danger)`,
          display:"flex", alignItems:"center", paddingLeft:6, gap:4,
          cursor:"pointer", zIndex: 3,
        }}
      >
        <IcAlert size={10} style={{color:"var(--danger)"}}/>
        <span className="mono" style={{fontSize:9.5, color:"var(--danger)", fontWeight:600, whiteSpace:"nowrap"}}>{block.label}</span>
      </div>
    );
  }

  // Transit block — main bar
  const s = block.shipment;
  const progressPx = Math.min(100, s.progress);
  return (
    <div
      onClick={onClick} onMouseEnter={onHover}
      data-tip={`${s.id} · ${s.customer} · ${s.eta}`}
      style={{
        position:"absolute", left:`${left}%`, width:`${width}%`,
        top: 10, bottom: 10,
        borderTopLeftRadius: clippedStart ? 0 : 5, borderBottomLeftRadius: clippedStart ? 0 : 5,
        borderTopRightRadius: clippedEnd ? 0 : 5, borderBottomRightRadius: clippedEnd ? 0 : 5,
        background: palette.bg,
        borderTop: `1px solid ${palette.bd}`,
        borderBottom: `1px solid ${palette.bd}`,
        borderLeft: clippedStart ? "none" : `1px solid ${palette.bd}`,
        borderRight: clippedEnd ? "none" : `1px solid ${palette.bd}`,
        boxShadow: highlighted ? `0 0 0 2px var(--brand-soft), 0 1px 2px rgba(0,0,0,0.05)` : expanded ? `0 0 0 2px var(--brand)` : "0 1px 2px rgba(0,0,0,0.04)",
        cursor:"pointer", overflow:"hidden",
        display:"flex", alignItems:"center", gap:6, padding:"0 8px",
        zIndex: highlighted ? 4 : 1,
        transition: "box-shadow 100ms ease",
      }}
    >
      {/* Inner progress fill — only on in-transit shipments */}
      {s.status !== "Unassigned" && s.status !== "Delivered" && (
        <div style={{
          position:"absolute", left:0, top:0, bottom:0,
          width:`${progressPx}%`,
          background:`linear-gradient(90deg, color-mix(in oklch, ${palette.bd} 22%, transparent), color-mix(in oklch, ${palette.bd} 12%, transparent))`,
          borderRight:`1.5px solid ${palette.bd}`,
        }}/>
      )}
      <div style={{position:"relative", display:"flex", alignItems:"center", gap:6, minWidth:0, flex:1}}>
        {clippedStart && <span style={{color:palette.bd, fontSize:11, fontWeight:700, marginLeft:-2}}>‹</span>}
        <span className="mono" style={{fontSize:10, fontWeight:600, color:palette.bd, whiteSpace:"nowrap"}}>{s.id.slice(-4)}</span>
        <span style={{fontSize:11, fontWeight:500, color:palette.fg, whiteSpace:"nowrap", overflow:"hidden", textOverflow:"ellipsis"}}>{s.customer}</span>
        {width > 12 && (
          <span className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", marginLeft:"auto", whiteSpace:"nowrap"}}>{s.miles}mi · ${s.revenue.toLocaleString()}</span>
        )}
        {clippedEnd && <span style={{color:palette.bd, fontSize:11, fontWeight:700, marginRight:-2}}>›</span>}
      </div>
      {/* Risk flag in corner */}
      {(s.status === "At Risk" || s.status === "Detention") && (
        <div style={{position:"absolute", top:-1, right:-1, width:0, height:0, borderTop:`8px solid var(--danger)`, borderLeft:"8px solid transparent"}}/>
      )}
    </div>
  );
}

function HosExhaustionBand({ driver, now, pct, end }) {
  // Parse "06:42" → hours
  const m = driver.hosLeft.match(/(\d+):(\d+)/);
  if (!m) return null;
  const hours = parseInt(m[1]) + parseInt(m[2]) / 60;
  if (hours > 6) return null; // only flag when getting tight
  const expiry = new Date(now.getTime() + hours * 3600 * 1000);
  if (expiry > end) return null;
  const left = pct(now);
  const right = pct(expiry);
  const tone = hours < 3 ? "var(--danger)" : "var(--warning)";
  return (
    <>
      <div style={{
        position:"absolute", left:`${left}%`, width:`${right - left}%`,
        top:0, bottom:0,
        background: `linear-gradient(90deg, transparent, color-mix(in oklch, ${tone} 14%, transparent))`,
        pointerEvents:"none",
      }}/>
      <div style={{
        position:"absolute", left:`${right}%`, top:2, bottom:2, width:0,
        borderLeft:`2px dashed ${tone}`, pointerEvents:"none",
      }}/>
      <div style={{
        position:"absolute", left:`calc(${right}% + 4px)`, top:4,
        fontSize:9, fontFamily:"'Geist Mono', monospace", color:tone, fontWeight:600,
        pointerEvents:"none", whiteSpace:"nowrap",
      }}>HOS 0:00</div>
    </>
  );
}

function ShiftBands({ hours, pct }) {
  // Paint very subtle night bands (22:00 – 06:00) so dispatchers see "off-hours"
  const bands = [];
  for (const h of hours) {
    if (h.getHours() === 22) {
      const next = new Date(h.getTime() + 8 * 3600 * 1000);
      bands.push({ from: h, to: next });
    }
  }
  return bands.map((b, i) => (
    <div key={i} style={{
      position:"absolute", top:0, bottom:0,
      left:`${pct(b.from)}%`, width:`${pct(b.to) - pct(b.from)}%`,
      background: "color-mix(in oklch, var(--fg) 3%, transparent)",
      pointerEvents:"none",
    }}/>
  ));
}

function HosBadge({ left }) {
  if (!left || left === "—") return <span style={{color:"var(--fg-subtle)"}}>HOS —</span>;
  const m = left.match(/(\d+):(\d+)/);
  const hours = m ? parseInt(m[1]) + parseInt(m[2]) / 60 : 12;
  const tone = hours < 3 ? "var(--danger)" : hours < 6 ? "var(--warning)" : "var(--fg-subtle)";
  return <span style={{color: tone, fontWeight: hours < 6 ? 600 : 400}}>HOS {left}</span>;
}

function DriverStatusDot({ status }) {
  const map = {
    moving:  { c:"var(--success)", t:"Moving" },
    dwell:   { c:"var(--danger)",  t:"Dwell" },
    loading: { c:"var(--info)",    t:"Loading" },
    unassigned:{c:"var(--fg-subtle)", t:"Idle" },
  };
  const m = map[status] || { c:"var(--fg-subtle)", t:"—" };
  return <span data-tip={m.t} className="dot" style={{background:m.c, color:m.c, width:7, height:7}}/>;
}

function fmtTime(d) {
  return d.toLocaleString("en-US", { month:"short", day:"numeric", hour:"2-digit", minute:"2-digit", hour12:false });
}

function TimelineLegend() {
  const items = [
    { c:"var(--success)", l:"On-time / delivered" },
    { c:"var(--brand)",   l:"In transit" },
    { c:"var(--info)",    l:"Loading / pickup" },
    { c:"var(--warning)", l:"At risk" },
    { c:"var(--danger)",  l:"Detention / late" },
    { c:"var(--fg-subtle)", l:"Unassigned" },
  ];
  return (
    <div style={{padding:"6px 12px", borderTop:"1px solid var(--border)", display:"flex", gap:14, alignItems:"center", flexWrap:"wrap"}}>
      <span className="label" style={{fontSize:9.5}}>Legend</span>
      {items.map((it, i) => (
        <span key={i} style={{display:"flex", alignItems:"center", gap:5, fontSize:10.5, color:"var(--fg-muted)"}}>
          <span style={{width:10, height:10, borderRadius:2, background:`color-mix(in oklch, ${it.c} 22%, var(--card))`, border:`1px solid ${it.c}`}}/>
          {it.l}
        </span>
      ))}
      <span style={{display:"flex", alignItems:"center", gap:5, fontSize:10.5, color:"var(--fg-muted)"}}>
        <span style={{width:14, height:8, borderRadius:1, background:`repeating-linear-gradient(45deg, color-mix(in oklch, var(--danger) 30%, transparent) 0 4px, transparent 4px 8px)`, border:"1px dashed var(--danger)"}}/>
        Dwell
      </span>
      <span style={{display:"flex", alignItems:"center", gap:5, fontSize:10.5, color:"var(--fg-muted)"}}>
        <span style={{width:2, height:14, background:"var(--brand)", boxShadow:"0 0 6px var(--brand-soft)"}}/>
        Now
      </span>
    </div>
  );
}

Object.assign(window, { TimelineView });
