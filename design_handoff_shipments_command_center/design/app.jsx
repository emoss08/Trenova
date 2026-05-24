/* Trenova Shipments Command Center — main app */

const { useState, useEffect, useRef, useMemo, useCallback } = React;

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "theme": "dark",
  "density": "cozy",
  "layout": "split",
  "showKpis": true,
  "mapStyle": "tactical",
  "timeWindow": "today",
  "accentHue": 263
}/*EDITMODE-END*/;

function App() {
  const [tweaks, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const [selectedView, setSelectedView] = useState("all");
  const [filters, setFilters] = useState([]);
  const [highlightId, setHighlightId] = useState(null);
  const [expandedId, setExpandedId] = useState("SHP-2026-1042");
  const [search, setSearch] = useState("");
  const [draggingId, setDraggingId] = useState(null);
  const [dropTargetDriver, setDropTargetDriver] = useState(null);
  const [recentlyAssigned, setRecentlyAssigned] = useState({});
  const [dismissedAlerts, setDismissedAlerts] = useState({});
  const [editingComment, setEditingComment] = useState(false);

  // Apply theme + accent to root
  useEffect(() => {
    document.documentElement.classList.toggle("dark", tweaks.theme === "dark");
    document.documentElement.style.setProperty("--brand", `oklch(${tweaks.theme === "dark" ? "0.62" : "0.55"} 0.22 ${tweaks.accentHue})`);
    document.documentElement.style.setProperty("--brand-soft", `oklch(${tweaks.theme === "dark" ? "0.62" : "0.55"} 0.22 ${tweaks.accentHue} / 0.14)`);
  }, [tweaks.theme, tweaks.accentHue]);

  const densityClass = `density-${tweaks.density}`;

  // Filter chips
  const toggleFilter = (f) => setFilters(p => p.includes(f) ? p.filter(x => x !== f) : [...p, f]);

  const filteredShipments = useMemo(() => {
    let s = SHIPMENTS;
    if (selectedView === "transit")  s = s.filter(x => x.status === "In Transit");
    if (selectedView === "atrisk")   s = s.filter(x => ["At Risk","Detention"].includes(x.status));
    if (selectedView === "unassign") s = s.filter(x => x.status === "Unassigned");
    if (selectedView === "detention")s = s.filter(x => x.status === "Detention");
    if (filters.includes("at-risk")) s = s.filter(x => ["At Risk","Detention"].includes(x.status));
    if (filters.includes("reefer"))  s = s.filter(x => /COLD|REF/.test(x.originCode + x.trailer));
    if (search) s = s.filter(x => (x.id + x.customer + x.origin + x.dest + x.driver).toLowerCase().includes(search.toLowerCase()));
    return s;
  }, [selectedView, filters, search]);

  return (
    <div className={densityClass} style={{ minHeight: "100vh" }}>
      <Sidebar tweaks={tweaks} setTweak={setTweak}/>
      <div style={{ marginLeft: 220, minHeight: "100vh", display: "flex", flexDirection: "column" }}>
        <TopBar tweaks={tweaks} setTweak={setTweak} search={search} setSearch={setSearch}/>
        <PageHeader tweaks={tweaks} setTweak={setTweak}/>

        {tweaks.showKpis && <KpiRail timeWindow={tweaks.timeWindow}/>}

        <div style={{ padding: "0 16px 12px", display: "grid", gap: 12, gridTemplateColumns:
            tweaks.layout === "map-first" ? "1fr 320px" :
            tweaks.layout === "table-first" ? "1fr 320px" :
            "1fr 340px",
            alignItems: "stretch"
        }}>
          {tweaks.layout === "map-first" ? (
            <MapPanel shipments={filteredShipments} drivers={DRIVERS} mapStyle={tweaks.mapStyle} setMapStyle={(v) => setTweak("mapStyle", v)} highlightId={highlightId} onHover={setHighlightId} onSelect={setExpandedId} large/>
          ) : tweaks.layout === "table-first" ? (
            <TablePanel shipments={filteredShipments} drivers={DRIVERS} expandedId={expandedId} setExpandedId={setExpandedId} highlightId={highlightId} onHover={setHighlightId} selectedView={selectedView} setSelectedView={setSelectedView} filters={filters} toggleFilter={toggleFilter} compact/>
          ) : (
            <MapPanel shipments={filteredShipments} drivers={DRIVERS} mapStyle={tweaks.mapStyle} setMapStyle={(v) => setTweak("mapStyle", v)} highlightId={highlightId} onHover={setHighlightId} onSelect={setExpandedId}/>
          )}

          <RightStack
            mapHeight={tweaks.layout === "map-first" ? 520 : 420}
            draggingId={draggingId} setDraggingId={setDraggingId}
            dropTargetDriver={dropTargetDriver} setDropTargetDriver={setDropTargetDriver}
            recentlyAssigned={recentlyAssigned} setRecentlyAssigned={setRecentlyAssigned}
            dismissedAlerts={dismissedAlerts} setDismissedAlerts={setDismissedAlerts}
          />
        </div>

        <div style={{ padding: "0 16px 12px" }}>
          <TablePanel shipments={filteredShipments} drivers={DRIVERS} expandedId={expandedId} setExpandedId={setExpandedId} highlightId={highlightId} onHover={setHighlightId} selectedView={selectedView} setSelectedView={setSelectedView} filters={filters} toggleFilter={toggleFilter}/>
        </div>

        <div style={{ padding: "0 16px 16px", display: "grid", gap: 12, gridTemplateColumns: "1fr 1fr 1fr" }}>
          <ActivityFeed/>
          <LaneHeatmap/>
          <CustomerMix/>
        </div>

        <Footer/>
      </div>

      <TweaksPanel title="Tweaks">
        <TweakSection title="Appearance">
          <TweakRadio label="Theme" value={tweaks.theme} onChange={(v) => setTweak("theme", v)} options={[{value:"light",label:"Light"},{value:"dark",label:"Dark"}]}/>
          <TweakRadio label="Density" value={tweaks.density} onChange={(v) => setTweak("density", v)} options={[{value:"compact",label:"Compact"},{value:"cozy",label:"Cozy"},{value:"comfortable",label:"Comfortable"}]}/>
          <TweakSlider label="Accent hue" value={tweaks.accentHue} onChange={(v) => setTweak("accentHue", v)} min={0} max={360} step={1}/>
        </TweakSection>
        <TweakSection title="Layout">
          <TweakRadio label="Workspace" value={tweaks.layout} onChange={(v) => setTweak("layout", v)} options={[{value:"split",label:"Split"},{value:"map-first",label:"Map-first"},{value:"table-first",label:"Table-first"}]}/>
          <TweakToggle label="Show KPI rail" value={tweaks.showKpis} onChange={(v) => setTweak("showKpis", v)}/>
        </TweakSection>
        <TweakSection title="Map">
          <TweakRadio label="Style" value={tweaks.mapStyle} onChange={(v) => setTweak("mapStyle", v)} options={[{value:"street",label:"Street"},{value:"satellite",label:"Satellite"},{value:"tactical",label:"Tactical"}]}/>
        </TweakSection>
        <TweakSection title="Time window">
          <TweakRadio label="Range" value={tweaks.timeWindow} onChange={(v) => setTweak("timeWindow", v)} options={[{value:"today",label:"Today"},{value:"24h",label:"24h"},{value:"7d",label:"7d"}]}/>
        </TweakSection>
      </TweaksPanel>
    </div>
  );
}

/* ---------------- chrome ---------------- */

function Sidebar({ tweaks }) {
  return (
    <aside style={{
      position:"fixed", left:0, top:0, bottom:0, width:220,
      background:"var(--bg-elev)", borderRight:"1px solid var(--border)",
      display:"flex", flexDirection:"column", padding:"10px 0", zIndex:5
    }}>
      <div style={{padding:"4px 14px 12px", display:"flex", alignItems:"center", gap:8}}>
        <div style={{
          width:22, height:22, borderRadius:5,
          background:"linear-gradient(135deg, var(--brand), oklch(0.5 0.2 200))",
          display:"flex", alignItems:"center", justifyContent:"center",
          color:"white", fontWeight:700, fontSize:12, letterSpacing:"-0.04em"
        }}>T</div>
        <div style={{display:"flex", flexDirection:"column", lineHeight:1.1}}>
          <div style={{fontWeight:600, fontSize:13, letterSpacing:"-0.01em"}}>Trenova</div>
          <div className="mono" style={{fontSize:9, color:"var(--fg-subtle)"}}>v2.0 · ops</div>
        </div>
      </div>

      <div style={{padding:"0 8px"}}>
        <div className="label" style={{padding:"10px 6px 4px"}}>Operations</div>
        <NavItem icon={<IcLayoutGrid size={14}/>} label="Dashboard"/>
        <NavItem icon={<IcTruck size={14}/>} label="Shipments" active count="142"/>
        <NavItem icon={<IcRoute size={14}/>} label="Dispatch" count="9"/>
        <NavItem icon={<IcUser size={14}/>} label="Drivers" count="38"/>
        <NavItem icon={<IcPin size={14}/>} label="Tracking"/>
        <NavItem icon={<IcShield size={14}/>} label="Compliance"/>

        <div className="label" style={{padding:"14px 6px 4px"}}>Billing</div>
        <NavItem icon={<IcDollar size={14}/>} label="Invoices"/>
        <NavItem icon={<IcFlag size={14}/>} label="Holds" count="3"/>
        <NavItem icon={<IcMessage size={14}/>} label="Statements"/>

        <div className="label" style={{padding:"14px 6px 4px"}}>Admin</div>
        <NavItem icon={<IcGear size={14}/>} label="Settings"/>
        <NavItem icon={<IcLayers size={14}/>} label="Integrations"/>
      </div>

      <div style={{marginTop:"auto", padding:"8px 10px", borderTop:"1px solid var(--border)"}}>
        <div style={{display:"flex", alignItems:"center", gap:8, padding:"4px 6px"}}>
          <div style={{width:24, height:24, borderRadius:"50%", background:"linear-gradient(135deg, oklch(0.7 0.18 30), oklch(0.6 0.20 320))", display:"flex", alignItems:"center", justifyContent:"center", color:"white", fontSize:10, fontWeight:600}}>SA</div>
          <div style={{flex:1, lineHeight:1.2}}>
            <div style={{fontSize:12, fontWeight:500}}>Sara Ahmed</div>
            <div className="mono" style={{fontSize:9, color:"var(--fg-subtle)"}}>Ops · Day shift</div>
          </div>
          <IcGear size={12} style={{color:"var(--fg-subtle)"}}/>
        </div>
      </div>
    </aside>
  );
}

function TopBar({ search, setSearch }) {
  const [now, setNow] = useState(new Date());
  useEffect(() => { const id = setInterval(() => setNow(new Date()), 1000); return () => clearInterval(id); }, []);
  const time = now.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
  return (
    <div style={{
      height:42, borderBottom:"1px solid var(--border)", padding:"0 14px",
      display:"flex", alignItems:"center", gap:12, position:"sticky", top:0, background:"color-mix(in oklch, var(--bg) 90%, transparent)", backdropFilter:"blur(8px)", zIndex:4
    }}>
      <div className="mono" style={{display:"flex", alignItems:"center", gap:6, fontSize:11, color:"var(--fg-muted)"}}>
        <span>Home</span><IcChevR size={10}/><span>Shipment Mgmt</span><IcChevR size={10}/><span style={{color:"var(--fg)"}}>Shipments</span>
      </div>
      <div style={{flex:1, display:"flex", justifyContent:"center"}}>
        <div style={{position:"relative", width:340}}>
          <IcSearch size={12} style={{position:"absolute", left:8, top:7, color:"var(--fg-subtle)"}}/>
          <input className="input mono" placeholder="Search shipments, customers, drivers, PROs…" value={search} onChange={(e)=>setSearch(e.target.value)} style={{width:"100%", paddingLeft:24, height:26, fontSize:11.5}}/>
          <kbd style={{position:"absolute", right:6, top:5}}>⌘K</kbd>
        </div>
      </div>
      <div className="mono" style={{fontSize:11, color:"var(--fg-muted)", display:"flex", gap:10, alignItems:"center"}}>
        <span style={{display:"flex", alignItems:"center", gap:4}}><span className="dot pulse" style={{background:"var(--success)", color:"var(--success)"}}/>System nominal</span>
        <span style={{color:"var(--fg-subtle)"}}>·</span>
        <span>{time} CT</span>
      </div>
      <div style={{display:"flex", gap:4}}>
        <button className="btn-ghost btn" data-tip="Keyboard shortcuts"><IcKb size={13}/></button>
        <button className="btn-ghost btn" data-tip="Notifications"><IcBell size={13}/></button>
      </div>
    </div>
  );
}

function PageHeader({ tweaks, setTweak }) {
  return (
    <div style={{padding:"14px 16px 10px", display:"flex", alignItems:"flex-end", justifyContent:"space-between", gap:16, flexWrap:"wrap"}}>
      <div>
        <div style={{display:"flex", alignItems:"center", gap:10}}>
          <h1 style={{margin:0, fontSize:22, fontWeight:600, letterSpacing:"-0.02em"}}>Shipments</h1>
          <span className="pill pill-soft-success" style={{padding:"2px 8px"}}>
            <span className="dot pulse" style={{background:"var(--success)", color:"var(--success)"}}/>Live · 142
          </span>
          <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>org · Trenova-DEV</span>
        </div>
        <div style={{fontSize:12, color:"var(--fg-muted)", marginTop:2}}>
          Operations command center — manage shipments, assignments, and exceptions.
        </div>
      </div>
      <div style={{display:"flex", gap:6, alignItems:"center"}}>
        <div className="card" style={{display:"flex", padding:2, gap:0}}>
          {[["today","Today"],["24h","24h"],["7d","7d"]].map(([v,l]) => (
            <button key={v} onClick={()=>setTweak("timeWindow", v)} className="btn" style={{
              border:"none", height:22, padding:"0 10px", fontSize:11, borderRadius:3,
              background: tweaks.timeWindow===v ? "var(--brand-soft)" : "transparent",
              color: tweaks.timeWindow===v ? "var(--brand)" : "var(--fg-muted)",
              fontWeight: tweaks.timeWindow===v ? 600 : 500
            }}>{l}</button>
          ))}
        </div>
        <button className="btn"><IcRefresh size={12}/>Refresh</button>
        <button className="btn"><IcDownload size={12}/>Export</button>
        <button className="btn"><IcEye size={12}/>Views</button>
        <button className="btn btn-primary"><IcPlus size={12}/>New shipment</button>
      </div>
    </div>
  );
}

/* ---------------- KPI rail ---------------- */

/* ---------------- KPI rail ---------------- */

function KpiRail({ timeWindow }) {
  return (
    <div style={{padding:"4px 16px 12px", display:"grid", gap:8, gridTemplateColumns:"repeat(12, minmax(0, 1fr))"}}>
      {/* Row 1: hero KPIs — bigger, sparkline-driven */}
      <KpiHero
        label="Revenue today" value="$36.4K" unit=""
        delta={8.1} deltaLabel="%" deltaTone="success"
        sub="RPM $2.18  ·  Avg margin 22.4%"
        sparkData={SERIES.revenue} sparkColor="var(--success)" icon={<IcDollar size={11}/>}
        span={3}
      />
      <KpiHero
        label="Active shipments" value="142"
        delta={5} deltaLabel="" deltaTone="success"
        sub="58 in-transit · 9 at-risk · 5 unassigned"
        sparkData={SERIES.active} sparkColor="var(--brand)" icon={<IcTruck size={11}/>}
        breakdown={[
          { label:"In-transit", value:58, color:"var(--brand)" },
          { label:"At-risk",   value:9,  color:"var(--danger)" },
          { label:"Loading",   value:6,  color:"var(--info)" },
          { label:"Done",      value:64, color:"var(--success)" },
        ]}
        span={3}
      />
      <KpiRing
        label="On-time"
        value="94.2" unit="%"
        target={96} ringValue={94.2} ringMax={100}
        delta={-1.2} deltaLabel="pp" deltaTone="danger"
        sub="Target 96%  ·  7-day 95.4%"
        icon={<IcClock size={11}/>}
        span={2}
      />
      <KpiGoalBar
        label="Empty mile %"
        value="11.8" unit="%"
        target={10} actual={11.8} max={20}
        delta={-0.4} deltaLabel="pp" deltaTone="success"
        sub="1,840 deadhead miles · goal <10%"
        icon={<IcRoute size={11}/>}
        span={2}
      />
      <KpiRing
        label="Tender accept"
        value="94.1" unit="%"
        target={95} ringValue={94.1} ringMax={100}
        delta={0.4} deltaLabel="pp" deltaTone="success"
        sub="23 accepted · 1 declined"
        icon={<IcCheck size={11}/>}
        span={2}
      />

      {/* Row 2: smaller status / watchlist KPIs */}
      <KpiStat label="At-risk" value="9" delta={1} tone="danger"
        sub="4 ETA slip · 3 weather · 2 reefer"
        icon={<IcAlert size={11}/>} span={2}/>
      <KpiStat label="Unassigned" value="5" delta={-2} tone="warning"
        sub="$8,650 revenue waiting"
        icon={<IcFlag size={11}/>} span={2}/>
      <KpiStat label="Ready to dispatch" value="12" delta={3} tone="brand"
        sub="5 unassigned · 7 driver-ready"
        icon={<IcBolt size={11}/>} span={2}/>
      <KpiWatchlist
        label="HOS near limit"
        items={[
          { who:"D-211 J. Park",      meta:"02:15 left", tone:"danger" },
          { who:"D-176 K. Whitehorse",meta:"04:30 left", tone:"warning" },
          { who:"D-189 L. Mendez",    meta:"07:48 left", tone:"muted" },
        ]}
        icon={<IcShield size={11}/>}
        span={3}
      />
      <KpiWatchlist
        label="Detention dwell > 2h"
        items={[
          { who:"SHP-1040 GlobalTrade", meta:"3h 38m", tone:"danger" },
          { who:"SHP-1041 FreshHaul",   meta:"2h 22m", tone:"warning" },
          { who:"SHP-1037 Peak",        meta:"2h 04m", tone:"warning" },
        ]}
        icon={<IcClock size={11}/>}
        span={3}
      />
    </div>
  );
}


/* ---------------- Map panel ---------------- */

function MapPanel({ shipments, drivers, mapStyle, setMapStyle, highlightId, onHover, onSelect, large }) {
  return (
    <div className="card" style={{ padding:0, height: large ? 520 : 420, display:"flex", flexDirection:"column", overflow:"hidden" }}>
      <div style={{display:"flex", alignItems:"center", justifyContent:"space-between", padding:"8px 12px", borderBottom:"1px solid var(--border)"}}>
        <div style={{display:"flex", alignItems:"center", gap:10}}>
          <div className="label">Live map</div>
          <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>contiguous US · 6 active units</span>
        </div>
        <div style={{display:"flex", gap:4}}>
          {[["tactical","Tactical"],["street","Street"],["satellite","Satellite"]].map(([v,l]) => (
            <button key={v} onClick={()=>setMapStyle(v)} className="btn" style={{
              height:22, padding:"0 8px", fontSize:10, fontWeight:500,
              background: mapStyle===v ? "var(--brand-soft)" : "transparent",
              color: mapStyle===v ? "var(--brand)" : "var(--fg-muted)",
              border: "1px solid " + (mapStyle===v ? "var(--brand)" : "var(--border)")
            }}>{l}</button>
          ))}
        </div>
      </div>
      <div style={{flex:1, position:"relative", padding:8}}>
        <ShipmentMap shipments={shipments} drivers={drivers} mapStyle={mapStyle} highlightId={highlightId} onHover={onHover} onSelect={onSelect}/>
      </div>
    </div>
  );
}

/* ---------------- Right stack: unassigned + exceptions + HOS + tomorrow ---------------- */

const RIGHT_PANEL_DEFS = [
  { id: "unassigned", title: "Unassigned" },
  { id: "exceptions", title: "Exceptions" },
  { id: "hos",        title: "HOS & dispatch" },
];

function RightStack({ mapHeight = 420, draggingId, setDraggingId, dropTargetDriver, setDropTargetDriver, recentlyAssigned, setRecentlyAssigned, dismissedAlerts, setDismissedAlerts }) {
  const [order, setOrder] = useState(() => {
    try { return JSON.parse(localStorage.getItem("trv_right_order")) || ["unassigned","exceptions","hos"]; } catch { return ["unassigned","exceptions","hos"]; }
  });
  const [hidden, setHidden] = useState(() => {
    try { return JSON.parse(localStorage.getItem("trv_right_hidden")) || []; } catch { return []; }
  });
  const [panelDrag, setPanelDrag] = useState(null);
  const [panelOver, setPanelOver] = useState(null);
  const [pickerOpen, setPickerOpen] = useState(false);

  useEffect(() => { localStorage.setItem("trv_right_order", JSON.stringify(order)); }, [order]);
  useEffect(() => { localStorage.setItem("trv_right_hidden", JSON.stringify(hidden)); }, [hidden]);

  const visible = order.filter(id => !hidden.includes(id));
  const hiddenList = RIGHT_PANEL_DEFS.filter(p => hidden.includes(p.id));

  const handleDrop = (targetId) => {
    if (!panelDrag || panelDrag === targetId) return;
    const next = order.filter(x => x !== panelDrag);
    const idx = next.indexOf(targetId);
    next.splice(idx, 0, panelDrag);
    setOrder(next);
    setPanelDrag(null);
    setPanelOver(null);
  };

  const hide = (id) => setHidden(h => [...new Set([...h, id])]);
  const show = (id) => setHidden(h => h.filter(x => x !== id));

  const renderPanel = (id) => {
    const isOver = panelOver === id;
    const dragHandlers = {
      onPanelDragStart: () => setPanelDrag(id),
      onPanelDragEnd: () => { setPanelDrag(null); setPanelOver(null); },
      onPanelDragOver: (e) => { if (panelDrag && panelDrag !== id) { e.preventDefault(); setPanelOver(id); } },
      onPanelDrop: (e) => { e.preventDefault(); handleDrop(id); },
      onHide: () => hide(id),
      isDropOver: isOver,
      isDragging: panelDrag === id,
    };
    if (id === "unassigned") return <UnassignedQueue key={id} draggingId={draggingId} setDraggingId={setDraggingId} recentlyAssigned={recentlyAssigned} {...dragHandlers}/>;
    if (id === "exceptions") return <ExceptionInbox key={id} dismissed={dismissedAlerts} setDismissed={setDismissedAlerts} {...dragHandlers}/>;
    if (id === "hos")        return <HosWatch key={id} setRecentlyAssigned={setRecentlyAssigned} draggingId={draggingId} setDropTargetDriver={setDropTargetDriver} dropTargetDriver={dropTargetDriver} setDraggingId={setDraggingId} {...dragHandlers}/>;
    return null;
  };

  const rows = visible.length > 0 ? `repeat(${visible.length}, 1fr)` : "1fr";

  return (
    <div style={{display:"grid", gridTemplateRows: rows, gap:10, height: mapHeight, minHeight: 0, position:"relative"}}>
      {visible.map(id => renderPanel(id))}
      {visible.length === 0 && (
        <div className="card" style={{display:"flex", alignItems:"center", justifyContent:"center", flexDirection:"column", gap:8, color:"var(--fg-muted)", fontSize:11.5, padding:20, textAlign:"center"}}>
          <span>All panels hidden.</span>
          <button className="btn btn-primary" onClick={() => setPickerOpen(true)}><IcPlus size={11}/>Add a panel</button>
        </div>
      )}
      {hiddenList.length > 0 && visible.length > 0 && (
        <div style={{position:"absolute", top:-26, right:0}}>
          <button className="btn" style={{height:22, fontSize:10.5, padding:"0 8px"}} onClick={() => setPickerOpen(p => !p)}>
            <IcPlus size={10}/>Add panel ({hiddenList.length})
          </button>
          {pickerOpen && (
            <div className="card fade-in" style={{position:"absolute", right:0, top:26, minWidth:180, padding:6, zIndex:10, boxShadow:"0 8px 24px -8px rgba(0,0,0,0.4)"}}>
              <div className="label" style={{padding:"4px 8px"}}>Hidden panels</div>
              {hiddenList.map(p => (
                <button key={p.id} onClick={() => { show(p.id); setPickerOpen(false); }}
                        style={{width:"100%", textAlign:"left", border:"none", background:"transparent", padding:"6px 8px", borderRadius:4, cursor:"pointer", fontSize:11.5, color:"var(--fg)", display:"flex", justifyContent:"space-between", alignItems:"center"}}
                        onMouseEnter={(e) => e.currentTarget.style.background = "color-mix(in oklch, var(--fg) 6%, transparent)"}
                        onMouseLeave={(e) => e.currentTarget.style.background = "transparent"}>
                  {p.title}<IcPlus size={10} style={{color:"var(--fg-subtle)"}}/>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Reusable header for the right-stack panels with drag handle + hide button.
function PanelHeader({ title, count, countTone = "muted", subtitle, action, onPanelDragStart, onPanelDragEnd, onHide }) {
  return (
    <div
      draggable
      onDragStart={(e) => { onPanelDragStart && onPanelDragStart(); e.dataTransfer.effectAllowed = "move"; try { e.dataTransfer.setData("text/plain", "panel"); } catch {} }}
      onDragEnd={onPanelDragEnd}
      style={{padding:"8px 10px 8px 8px", borderBottom:"1px solid var(--border)", display:"flex", justifyContent:"space-between", alignItems:"center", flexShrink:0, cursor:"grab", userSelect:"none"}}>
      <div style={{display:"flex", alignItems:"center", gap:6, minWidth:0}}>
        <IcGrip size={12} style={{color:"var(--fg-subtle)", flexShrink:0}}/>
        <div className="label" style={{whiteSpace:"nowrap"}}>{title}</div>
        {count !== undefined && <span className={`pill pill-soft-${countTone}`}>{count}</span>}
        {subtitle && <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)", whiteSpace:"nowrap", overflow:"hidden", textOverflow:"ellipsis"}}>{subtitle}</span>}
      </div>
      <div style={{display:"flex", alignItems:"center", gap:4, flexShrink:0}}>
        {action}
        <button className="btn-ghost btn" onClick={onHide} data-tip="Hide panel" style={{height:20, width:20, padding:0, justifyContent:"center", color:"var(--fg-subtle)"}}>
          <svg width="10" height="10" viewBox="0 0 10 10"><path d="M2 5h6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round"/></svg>
        </button>
      </div>
    </div>
  );
}

function UnassignedQueue({ draggingId, setDraggingId, recentlyAssigned, onPanelDragStart, onPanelDragEnd, onPanelDragOver, onPanelDrop, onHide, isDropOver, isDragging }) {
  return (
    <div className="card" onDragOver={onPanelDragOver} onDrop={onPanelDrop}
         style={{display:"flex", flexDirection:"column", minHeight:0, overflow:"hidden",
                 outline: isDropOver ? "2px dashed var(--brand)" : "none", outlineOffset:-2,
                 opacity: isDragging ? 0.4 : 1, transition:"opacity 100ms"}}>
      <PanelHeader title="Unassigned" count={UNASSIGNED.length} countTone="brand" subtitle="$8,650 waiting"
        onPanelDragStart={onPanelDragStart} onPanelDragEnd={onPanelDragEnd} onHide={onHide}/>
      <div style={{padding:8, display:"flex", flexDirection:"column", gap:6, overflowY:"auto", flex:1, minHeight:0}}>
        {UNASSIGNED.filter(u => !recentlyAssigned[u.id]).map(u => (
          <div key={u.id} className="draggable" draggable
               onDragStart={(e)=>{ setDraggingId(u.id); e.dataTransfer.effectAllowed = "move"; }}
               onDragEnd={()=>setDraggingId(null)}
               style={{
                 background: draggingId === u.id ? "var(--brand-soft)" : "var(--card-2)",
                 border:"1px solid var(--border)", borderRadius:5, padding:"7px 9px",
                 display:"flex", flexDirection:"column", gap:4, fontSize:11,
                 transition: "background 80ms"
               }}>
            <div style={{display:"flex", justifyContent:"space-between", alignItems:"center"}}>
              <div style={{display:"flex", alignItems:"center", gap:6}}>
                <IcGrip size={11} style={{color:"var(--fg-subtle)"}}/>
                <span className="mono tabular" style={{fontWeight:600, fontSize:11}}>{u.lane}</span>
              </div>
              {u.priority === "high" && <span className="pill pill-soft-danger">HOT</span>}
              {u.priority === "med"  && <span className="pill pill-soft-warning">MED</span>}
              {u.priority === "low"  && <span className="pill pill-soft-muted">LOW</span>}
            </div>
            <div style={{display:"flex", justifyContent:"space-between", alignItems:"center", color:"var(--fg-muted)", fontSize:10.5}}>
              <span>{u.customer}</span>
              <span className="mono tabular">{u.equip}</span>
            </div>
            <div style={{display:"flex", justifyContent:"space-between", alignItems:"baseline"}}>
              <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>pickup {u.pickup}</span>
              <span className="mono tabular" style={{fontSize:11.5, fontWeight:600}}>${u.revenue.toLocaleString()} <span style={{color:"var(--fg-subtle)", fontWeight:400}}>· {u.miles}mi</span></span>
            </div>
          </div>
        ))}
        {Object.keys(recentlyAssigned).length > 0 && (
          <div className="mono" style={{fontSize:10, color:"var(--success)", textAlign:"center", padding:4, background:"var(--success-soft)", borderRadius:4}}>
            ✓ {Object.keys(recentlyAssigned).length} assigned this session
          </div>
        )}
      </div>
      <div style={{padding:"6px 12px", borderTop:"1px solid var(--border)", fontSize:10, color:"var(--fg-subtle)", display:"flex", justifyContent:"space-between"}}>
        <span>Drag to a driver below to assign</span>
        <span className="mono">↕ reorder</span>
      </div>
    </div>
  );
}

// Wrapper that gives a card a flexible scroll body sized to its grid track.
function ScrollCard({ children }) {
  return <div className="card" style={{display:"flex", flexDirection:"column", minHeight:0, overflow:"hidden"}}>{children}</div>;
}

function ExceptionInbox({ dismissed, setDismissed, onPanelDragStart, onPanelDragEnd, onPanelDragOver, onPanelDrop, onHide, isDropOver, isDragging }) {
  const items = [
    { id:"e1", sev:"danger",  icon:<IcThermo size={12}/>, title:"Reefer temp drift", body:"SHP-2026-1041 · −2°F vs setpoint", time:"4 min", action:"Notify driver" },
    { id:"e2", sev:"warning", icon:<IcClock size={12}/>, title:"Detention 3h 38m", body:"SHP-2026-1040 at GlobalTrade DC", time:"12 min", action:"Bill detention" },
    { id:"e3", sev:"danger",  icon:<IcAlert size={12}/>, title:"ETA slip 2h",      body:"SHP-2026-1041 · weather Albuquerque", time:"22 min", action:"Re-quote ETA" },
    { id:"e4", sev:"warning", icon:<IcShield size={12}/>, title:"HOS exhaustion",   body:"D-211 J. Park · 02:15 left", time:"38 min", action:"Force break" },
  ].filter(i => !dismissed[i.id]);
  return (
    <div className="card" onDragOver={onPanelDragOver} onDrop={onPanelDrop}
         style={{display:"flex", flexDirection:"column", minHeight:0, overflow:"hidden",
                 outline: isDropOver ? "2px dashed var(--brand)" : "none", outlineOffset:-2,
                 opacity: isDragging ? 0.4 : 1, transition:"opacity 100ms"}}>
      <PanelHeader title="Exceptions" count={items.length} countTone="danger"
        action={<button className="btn-ghost btn" style={{height:20, padding:"0 6px", fontSize:10}}>Mute · 1h</button>}
        onPanelDragStart={onPanelDragStart} onPanelDragEnd={onPanelDragEnd} onHide={onHide}/>
      <div style={{display:"flex", flexDirection:"column", overflowY:"auto", flex:1, minHeight:0}}>
        {items.map((it, i) => {
          const color = it.sev === "danger" ? "var(--danger)" : "var(--warning)";
          return (
            <div key={it.id} style={{padding:"8px 12px", borderTop: i>0 ? "1px solid var(--border-2)" : "none", display:"flex", gap:8}}>
              <div style={{width:22, height:22, borderRadius:4, background: it.sev==="danger" ? "var(--danger-soft)" : "var(--warning-soft)", color, display:"flex", alignItems:"center", justifyContent:"center", flexShrink:0}}>
                {it.icon}
              </div>
              <div style={{flex:1, minWidth:0}}>
                <div style={{display:"flex", justifyContent:"space-between", alignItems:"baseline"}}>
                  <span style={{fontSize:11.5, fontWeight:600}}>{it.title}</span>
                  <span className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{it.time}</span>
                </div>
                <div style={{fontSize:10.5, color:"var(--fg-muted)", marginTop:1}}>{it.body}</div>
                <div style={{display:"flex", gap:6, marginTop:5}}>
                  <button className="btn" style={{height:20, fontSize:10, padding:"0 7px", color, borderColor: color}}>{it.action}</button>
                  <button className="btn-ghost btn" onClick={()=>setDismissed(d => ({...d, [it.id]: true}))} style={{height:20, fontSize:10, padding:"0 6px"}}>Dismiss</button>
                </div>
              </div>
            </div>
          );
        })}
        {items.length === 0 && <div style={{padding:16, textAlign:"center", fontSize:11, color:"var(--fg-subtle)"}}>All clear ✓</div>}
      </div>
    </div>
  );
}

function HosWatch({ draggingId, setDraggingId, dropTargetDriver, setDropTargetDriver, setRecentlyAssigned, onPanelDragStart, onPanelDragEnd, onPanelDragOver, onPanelDrop, onHide, isDropOver, isDragging }) {
  return (
    <div className="card" onDragOver={onPanelDragOver} onDrop={onPanelDrop}
         style={{display:"flex", flexDirection:"column", minHeight:0, overflow:"hidden",
                 outline: isDropOver ? "2px dashed var(--brand)" : "none", outlineOffset:-2,
                 opacity: isDragging ? 0.4 : 1, transition:"opacity 100ms"}}>
      <PanelHeader title="HOS & dispatch"
        action={<span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>{DRIVERS.length} on duty</span>}
        onPanelDragStart={onPanelDragStart} onPanelDragEnd={onPanelDragEnd} onHide={onHide}/>
      <div style={{display:"flex", flexDirection:"column", overflowY:"auto", flex:1, minHeight:0}}>
        {DRIVERS.slice(0, 5).map((d, i) => {
          const [h, m] = d.hosLeft.split(":").map(Number);
          const minLeft = h * 60 + m;
          const sev = minLeft < 180 ? "danger" : minLeft < 300 ? "warning" : "success";
          const sevColor = sev === "danger" ? "var(--danger)" : sev === "warning" ? "var(--warning)" : "var(--success)";
          const pct = Math.max(8, Math.min(100, (minLeft / 660) * 100));
          const isDrop = dropTargetDriver === d.id;
          return (
            <div key={d.id}
                 className={`row drop-target ${isDrop ? "is-over" : ""}`}
                 onDragOver={(e)=>{ e.preventDefault(); setDropTargetDriver(d.id); }}
                 onDragLeave={()=> setDropTargetDriver(prev => prev === d.id ? null : prev)}
                 onDrop={(e)=>{
                   e.preventDefault();
                   if (draggingId) {
                     setRecentlyAssigned(r => ({...r, [draggingId]: d.id}));
                     setDraggingId(null);
                     setDropTargetDriver(null);
                   }
                 }}
                 style={{padding:"7px 12px", borderTop: i>0 ? "1px solid var(--border-2)" : "none", display:"flex", flexDirection:"column", gap:4}}>
              <div style={{display:"flex", alignItems:"center", gap:8}}>
                <div style={{width:22, height:22, borderRadius:"50%", background:"linear-gradient(135deg, oklch(0.65 0.16 200), oklch(0.6 0.18 280))", display:"flex", alignItems:"center", justifyContent:"center", color:"white", fontSize:9, fontWeight:600}}>
                  {d.name.split(" ").map(p => p[0]).join("")}
                </div>
                <div style={{flex:1, minWidth:0}}>
                  <div style={{fontSize:11.5, fontWeight:500, display:"flex", alignItems:"center", gap:6}}>
                    {d.name}
                    <span className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{d.tractor}</span>
                  </div>
                  <div className="mono" style={{fontSize:10, color:"var(--fg-muted)"}}>{d.lane}</div>
                </div>
                <div style={{textAlign:"right"}}>
                  <div className="mono tabular" style={{fontSize:11.5, fontWeight:600, color: sevColor}}>{d.hosLeft}</div>
                  <div className="mono" style={{fontSize:9, color:"var(--fg-subtle)", textTransform:"uppercase", letterSpacing:"0.04em"}}>HOS left</div>
                </div>
              </div>
              <div className="lane-bar" style={{marginTop:2}}>
                <span style={{width: `${pct}%`, background: sevColor}}/>
              </div>
            </div>
          );
        })}
      </div>
      <div style={{padding:"6px 12px", borderTop:"1px solid var(--border)", fontSize:10, color:"var(--fg-subtle)", display:"flex", justifyContent:"space-between", flexShrink:0}}>
        <span>Drop a load on a driver to assign</span>
        <a href="#" style={{color:"var(--brand)"}}>View all drivers →</a>
      </div>
    </div>
  );
}

/* ---------------- Table panel ---------------- */

function TablePanel({ shipments, drivers, expandedId, setExpandedId, highlightId, onHover, selectedView, setSelectedView, filters, toggleFilter, compact }) {
  const [viewMode, setViewMode] = React.useState(() => {
    try { return localStorage.getItem("trenova.viewMode") || "timeline"; } catch { return "timeline"; }
  });
  const setMode = (m) => { setViewMode(m); try { localStorage.setItem("trenova.viewMode", m); } catch {} };
  return (
    <div className="card">
      <ViewSegments selectedView={selectedView} setSelectedView={setSelectedView} viewMode={viewMode} setViewMode={setMode}/>
      <FilterRow filters={filters} toggleFilter={toggleFilter} count={shipments.length}/>
      {viewMode === "timeline" ? (
        <TimelineView shipments={shipments} drivers={drivers || []} expandedId={expandedId} setExpandedId={setExpandedId} highlightId={highlightId} onHover={onHover}/>
      ) : (
        <ShipmentTable shipments={shipments} expandedId={expandedId} setExpandedId={setExpandedId} highlightId={highlightId} onHover={onHover}/>
      )}
      <TableFooter count={shipments.length}/>
    </div>
  );
}

function ViewSegments({ selectedView, setSelectedView, viewMode, setViewMode }) {
  return (
    <div style={{padding:"6px 10px", borderBottom:"1px solid var(--border)", display:"flex", gap:2, overflowX:"auto", alignItems:"center"}} className="no-scrollbar">
      {SAVED_VIEWS.map(v => {
        const active = selectedView === v.id;
        return (
          <button key={v.id} onClick={() => setSelectedView(v.id)} style={{
            border:"none", background:"transparent",
            padding:"5px 10px", borderRadius:4, cursor:"pointer",
            display:"flex", alignItems:"center", gap:6,
            fontSize:11.5, fontWeight: active ? 600 : 500,
            color: active ? "var(--fg)" : "var(--fg-muted)",
            position:"relative"
          }}>
            {v.label}
            <span className="mono tabular" style={{fontSize:9.5, color: active ? "var(--brand)" : "var(--fg-subtle)", background: active ? "var(--brand-soft)" : "color-mix(in oklch, var(--fg) 6%, transparent)", padding:"1px 5px", borderRadius:3}}>{v.count}</span>
            {active && <span style={{position:"absolute", left:4, right:4, bottom:-7, height:2, background:"var(--brand)", borderRadius:2}}/>}
          </button>
        );
      })}
      <div style={{marginLeft:"auto", display:"flex", gap:6, alignItems:"center"}}>
        {viewMode !== undefined && (
          <div style={{display:"flex", gap:2, background:"var(--card-2)", border:"1px solid var(--border)", borderRadius:4, padding:2}}>
            {[
              { id:"table",    label:"Table",    icon:<IcLayout size={11}/> },
              { id:"timeline", label:"Timeline", icon:<IcRoute size={11}/> },
            ].map(m => {
              const on = viewMode === m.id;
              return (
                <button key={m.id} onClick={() => setViewMode(m.id)} style={{
                  border:"none", background: on ? "var(--card)" : "transparent",
                  color: on ? "var(--fg)" : "var(--fg-muted)", fontWeight: on ? 600 : 500,
                  padding:"3px 8px", borderRadius:3, fontSize:11, cursor:"pointer",
                  display:"flex", alignItems:"center", gap:4,
                  boxShadow: on ? "0 1px 0 var(--border)" : "none",
                }}>{m.icon}{m.label}</button>
              );
            })}
          </div>
        )}
        <button className="btn-ghost btn" style={{height:24, padding:"0 8px", fontSize:11, color:"var(--fg-muted)"}}><IcPlus size={11}/>Save view</button>
      </div>
    </div>
  );
}

function FilterRow({ filters, toggleFilter, count }) {
  const chips = [
    { id:"at-risk", label:"Status: at-risk", color:"danger" },
    { id:"reefer",  label:"Equipment: reefer", color:"brand" },
    { id:"today",   label:"Delivering today", color:"muted" },
    { id:"hot",     label:"Priority: hot", color:"warning" },
  ];
  return (
    <div style={{padding:"6px 10px", borderBottom:"1px solid var(--border)", display:"flex", gap:6, alignItems:"center"}}>
      <button className="btn"><IcFilter size={11}/>Filter</button>
      <button className="btn"><IcSort size={11}/>Sort</button>
      <div style={{width:1, height:18, background:"var(--border)", margin:"0 2px"}}/>
      <div style={{display:"flex", gap:4, flex:1, flexWrap:"wrap"}}>
        {chips.map(c => {
          const on = filters.includes(c.id);
          return (
            <button key={c.id} onClick={() => toggleFilter(c.id)} className={`pill pill-soft-${on ? c.color : "muted"}`} style={{
              cursor:"pointer", border: on ? `1px solid color-mix(in oklch, var(--${c.color === "muted" ? "fg" : c.color}) 30%, transparent)` : "1px solid transparent",
              padding:"3px 8px", height:22
            }}>
              {c.label}{on && " ×"}
            </button>
          );
        })}
      </div>
      <span className="mono" style={{fontSize:10.5, color:"var(--fg-muted)"}}>{count} of 142 results</span>
      <button className="btn"><IcLayout size={11}/>Columns</button>
    </div>
  );
}

function ShipmentTable({ shipments, expandedId, setExpandedId, highlightId, onHover }) {
  return (
    <div style={{overflowX:"auto"}}>
      <table style={{width:"100%", borderCollapse:"collapse", minWidth:1100}}>
        <thead>
          <tr style={{borderBottom:"1px solid var(--border)"}}>
            {[
              {label:"Lane", w:"20%"},
              {label:"Status", w:"14%"},
              {label:"PRO / BOL", w:"11%"},
              {label:"Customer", w:"13%"},
              {label:"Driver / Equip", w:"13%"},
              {label:"ETA", w:"10%"},
              {label:"Revenue", w:"10%", align:"right"},
              {label:"Margin", w:"7%", align:"right"},
              {label:"", w:"2%"},
            ].map((c,i) => (
              <th key={i} className="label" style={{textAlign: c.align || "left", padding:"6px 10px", width:c.w, fontSize:9.5, color:"var(--fg-subtle)"}}>{c.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {shipments.map((s, idx) => (
            <ShipmentRow key={s.id} s={s} idx={idx} expanded={expandedId === s.id} highlighted={highlightId === s.id}
                         onClick={() => setExpandedId(expandedId === s.id ? null : s.id)}
                         onHover={onHover}/>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ShipmentRow({ s, idx, expanded, highlighted, onClick, onHover }) {
  const laneClass =
    s.status === "Delivered"  ? "is-done" :
    s.status === "At Risk"    ? "is-late" :
    s.status === "Detention"  ? "is-warn" : "";
  const etaColor =
    s.etaStatus === "late"     ? "var(--danger)"  :
    s.etaStatus === "watch"    ? "var(--warning)" :
    s.etaStatus === "delivered"? "var(--success)" :
    s.etaStatus === "pending"  ? "var(--fg-subtle)" : "var(--fg)";
  return (
    <>
      <tr onClick={onClick} onMouseEnter={() => onHover(s.id)} onMouseLeave={() => onHover(null)}
          className={`row ${expanded ? "is-expanded" : ""} ${highlighted && !expanded ? "is-highlighted" : ""}`}
          style={{ borderBottom: "1px solid var(--border-2)", height: "var(--row-h)" }}>
        <td style={{padding: "var(--pad-y) var(--pad-x)"}}>
          <div style={{display:"flex", alignItems:"center", gap:8}}>
            <span className="mono tabular" style={{fontSize:11.5, fontWeight:600, color: highlighted ? "var(--brand)" : "var(--fg)"}}>{s.originCode}</span>
            <span style={{color:"var(--fg-subtle)"}}>→</span>
            <span className="mono tabular" style={{fontSize:11.5, fontWeight:600}}>{s.destCode}</span>
            <div className={`lane-bar ${laneClass}`} style={{flex:1, maxWidth:80, marginLeft:6}}>
              <span style={{width: `${s.progress}%`}}/>
            </div>
            <span className="mono tabular" style={{fontSize:9.5, color:"var(--fg-subtle)", minWidth:28, textAlign:"right"}}>{s.progress}%</span>
          </div>
          <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", marginTop:1}}>{s.miles.toLocaleString()}mi · {s.commodity}</div>
        </td>
        <td style={{padding: "var(--pad-y) var(--pad-x)"}}>
          <div style={{display:"flex", alignItems:"center", gap:6}}>
            <StatusPill status={s.status}/>
            {s.lastEvent && <span className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", whiteSpace:"nowrap", overflow:"hidden", textOverflow:"ellipsis", maxWidth:100}} data-tip={s.lastEvent}>{s.lastEventAt}</span>}
          </div>
        </td>
        <td className="mono tabular" style={{padding: "var(--pad-y) var(--pad-x)", fontSize:11}}>
          <div style={{fontWeight:500}}>{s.pro}</div>
          <div style={{color:"var(--fg-subtle)", fontSize:10}}>{s.bol}</div>
        </td>
        <td style={{padding: "var(--pad-y) var(--pad-x)", fontSize:11.5}}>
          <div>{s.customer}</div>
          <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{s.weight}</div>
        </td>
        <td style={{padding: "var(--pad-y) var(--pad-x)", fontSize:11.5}}>
          {s.driver === "—" ? (
            <span style={{color:"var(--warning)", fontSize:11, display:"flex", alignItems:"center", gap:4}}><IcAlert size={11}/>Needs driver</span>
          ) : (
            <>
              <div style={{fontWeight:500}}>{s.driver}</div>
              <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{s.tractor} · {s.trailer}</div>
            </>
          )}
        </td>
        <td className="mono tabular" style={{padding: "var(--pad-y) var(--pad-x)", fontSize:11}}>
          <div style={{color: etaColor, fontWeight:500}}>{s.eta}</div>
          {s.hosLeft !== "—" && <div style={{fontSize:9.5, color:"var(--fg-subtle)"}}>HOS {s.hosLeft}</div>}
        </td>
        <td className="mono tabular" style={{padding: "var(--pad-y) var(--pad-x)", textAlign:"right", fontSize:11.5, fontWeight:600}}>
          ${s.revenue.toLocaleString()}
          <div style={{fontSize:9.5, color:"var(--fg-subtle)", fontWeight:400}}>${s.rpm}/mi</div>
        </td>
        <td className="mono tabular" style={{padding: "var(--pad-y) var(--pad-x)", textAlign:"right", fontSize:11.5}}>
          <span style={{color: s.margin >= 20 ? "var(--success)" : s.margin >= 15 ? "var(--warning)" : "var(--danger)", fontWeight:600}}>{s.margin}%</span>
        </td>
        <td style={{padding: "var(--pad-y) 6px"}}>
          <button className="btn-ghost btn" style={{height:20, padding:"0 4px"}} onClick={(e) => { e.stopPropagation(); }}><IcDots size={12}/></button>
        </td>
      </tr>
      {expanded && <ExpandedRow s={s}/>}
    </>
  );
}

function ExpandedRow({ s }) {
  return (
    <tr className="fade-in" style={{background:"color-mix(in oklch, var(--brand) 3%, var(--card))"}}>
      <td colSpan={9} style={{padding:"10px 14px", borderBottom:"1px solid var(--border)"}}>
        <div style={{display:"grid", gridTemplateColumns:"2fr 1.4fr 1fr 1fr", gap:18}}>
          {/* Stops timeline */}
          <div>
            <div className="label" style={{marginBottom:6}}>Route timeline</div>
            <div style={{position:"relative", paddingLeft:16}}>
              <div style={{position:"absolute", left:5, top:8, bottom:8, width:1.5, background:"var(--border)"}}/>
              {[
                { t:"Apr 21, 08:00", k:"PICKUP", loc:s.origin, status:"done", note:"Loaded · sealed S-44291" },
                { t:"Apr 21, 14:30", k:"FUEL",   loc:"Pilot #482, Albuquerque", status:"done", note:"212 gal · $3.84" },
                { t:"Apr 22, 02:15", k:"REST",   loc:"Amarillo, TX", status:"done", note:"10h reset" },
                { t:"now",           k:"IN-TRANSIT", loc:"~Kansas City line", status:"current", note:s.lastEvent },
                { t:"Apr 22, 14:30", k:"DELIVERY", loc:s.dest, status:"upcoming", note:"Live unload · door 4" },
              ].map((stp, i) => (
                <div key={i} style={{position:"relative", paddingBottom:8, fontSize:11}}>
                  <div style={{position:"absolute", left:-13, top:3, width:9, height:9, borderRadius:"50%",
                    background: stp.status==="current" ? "var(--brand)" : stp.status==="done" ? "var(--success)" : "var(--card-2)",
                    border: stp.status==="upcoming" ? "1.5px dashed var(--border)" : "1.5px solid var(--card)",
                    boxShadow: stp.status==="current" ? "0 0 0 3px var(--brand-soft)" : "none"
                  }}/>
                  <div style={{display:"flex", alignItems:"baseline", gap:8}}>
                    <span className="mono tabular" style={{fontSize:10, color:"var(--fg-subtle)", width:88}}>{stp.t}</span>
                    <span className="mono" style={{fontSize:9.5, fontWeight:600, color: stp.status==="current" ? "var(--brand)" : "var(--fg-muted)", letterSpacing:"0.06em"}}>{stp.k}</span>
                    <span style={{fontWeight:500}}>{stp.loc}</span>
                  </div>
                  <div style={{fontSize:10.5, color:"var(--fg-muted)", marginLeft:96}}>{stp.note}</div>
                </div>
              ))}
            </div>
          </div>

          {/* Financial breakdown */}
          <div>
            <div className="label" style={{marginBottom:6}}>Financials</div>
            <div style={{display:"grid", gridTemplateColumns:"1fr 1fr", gap:6, fontSize:11}}>
              {[
                ["Linehaul", `$${(s.revenue * 0.82).toFixed(0)}`],
                ["Fuel surcharge", `$${(s.revenue * 0.13).toFixed(0)}`],
                ["Accessorials", `$${(s.revenue * 0.05).toFixed(0)}`],
                ["Total revenue", `$${s.revenue.toLocaleString()}`, true],
                ["Driver pay", `$${(s.revenue * 0.42).toFixed(0)}`],
                ["Fuel cost", `$${(s.revenue * 0.21).toFixed(0)}`],
                ["Tolls & misc", `$${(s.revenue * 0.04).toFixed(0)}`],
                ["Margin", `${s.margin}%`, true, s.margin >= 20 ? "var(--success)" : "var(--warning)"],
              ].map(([k, v, bold, color], i) => (
                <div key={i} style={{display:"flex", justifyContent:"space-between", padding:"3px 0", borderTop: bold ? "1px solid var(--border-2)" : "none", marginTop: bold ? 2 : 0}}>
                  <span style={{color:"var(--fg-muted)"}}>{k}</span>
                  <span className="mono tabular" style={{fontWeight: bold ? 600 : 500, color: color || "var(--fg)"}}>{v}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Documents */}
          <div>
            <div className="label" style={{marginBottom:6}}>Documents</div>
            <div style={{display:"flex", flexDirection:"column", gap:4, fontSize:11}}>
              {[
                ["Rate confirmation", "RC-1042.pdf", "var(--success)"],
                ["BOL", `${s.bol}.pdf`, "var(--success)"],
                ["Lumper receipt", "—", "var(--fg-subtle)"],
                ["POD", "Pending", "var(--warning)"],
                ["Customer invoice", "—", "var(--fg-subtle)"],
              ].map(([k, v, c], i) => (
                <div key={i} style={{display:"flex", justifyContent:"space-between", alignItems:"center"}}>
                  <span style={{color:"var(--fg-muted)"}}>{k}</span>
                  <span className="mono tabular" style={{color: c, fontSize:10.5}}>{v}</span>
                </div>
              ))}
            </div>
            <button className="btn" style={{marginTop:8, width:"100%", height:24, fontSize:11}}><IcPlus size={11}/>Upload</button>
          </div>

          {/* Quick actions + comments */}
          <div>
            <div className="label" style={{marginBottom:6}}>Quick actions</div>
            <div style={{display:"flex", flexDirection:"column", gap:5, fontSize:11}}>
              <button className="btn"><IcMessage size={11}/>Message driver</button>
              <button className="btn"><IcPin size={11}/>Update ETA</button>
              <button className="btn"><IcDollar size={11}/>Add accessorial</button>
              <button className="btn" style={{color:"var(--danger)", borderColor:"color-mix(in oklch, var(--danger) 30%, var(--border))"}}><IcAlert size={11}/>Cancel shipment</button>
            </div>
            <div className="label" style={{marginTop:10, marginBottom:4}}>Comments · 3</div>
            <div style={{background:"var(--card-2)", border:"1px solid var(--border)", borderRadius:4, padding:6, fontSize:10.5, lineHeight:1.4, color:"var(--fg-muted)"}}>
              <div style={{marginBottom:3}}><strong style={{color:"var(--fg)"}}>@m.diaz</strong> 12m · Confirmed pickup with shipper, eyes on detention.</div>
              <div style={{marginBottom:3}}><strong style={{color:"var(--fg)"}}>@k.tan</strong> 1h · Got rate at $4,280 — slight bump from quote.</div>
              <div><strong style={{color:"var(--fg)"}}>@b.huang</strong> 3h · Driver M. Alvarez assigned — preferred for Acme.</div>
            </div>
          </div>
        </div>
      </td>
    </tr>
  );
}

function TableFooter({ count }) {
  return (
    <div style={{padding:"6px 12px", borderTop:"1px solid var(--border)", display:"flex", justifyContent:"space-between", alignItems:"center", fontSize:11, color:"var(--fg-muted)"}}>
      <div className="mono">{count} rows · 25 of 142 shown</div>
      <div style={{display:"flex", gap:4, alignItems:"center"}}>
        <button className="btn-ghost btn" style={{height:22, padding:"0 6px"}}><IcChevR size={11} style={{transform:"rotate(180deg)"}}/></button>
        <span className="mono">1 / 6</span>
        <button className="btn-ghost btn" style={{height:22, padding:"0 6px"}}><IcChevR size={11}/></button>
      </div>
    </div>
  );
}

/* ---------------- Activity feed ---------------- */

function ActivityFeed() {
  const [open, setOpen] = useState(true);
  return (
    <div className="card" style={{display:"flex", flexDirection:"column"}}>
      <div style={{padding:"8px 12px", borderBottom:"1px solid var(--border)", display:"flex", justifyContent:"space-between", alignItems:"center"}}>
        <div style={{display:"flex", alignItems:"center", gap:8}}>
          <div className="label">Activity stream</div>
          <span className="dot pulse" style={{background:"var(--success)", color:"var(--success)"}}/>
          <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>live</span>
        </div>
        <button className="btn-ghost btn" style={{height:20, fontSize:10, padding:"0 6px"}}>Filter</button>
      </div>
      <div style={{padding:8, display:"flex", flexDirection:"column", gap:0, maxHeight:240, overflowY:"auto"}}>
        {ACTIVITY.map((a, i) => {
          const sev = a.sev === "danger" ? "var(--danger)" : a.sev === "success" ? "var(--success)" : a.sev === "brand" ? "var(--brand)" : a.sev === "info" ? "var(--info)" : "var(--fg-subtle)";
          return (
            <div key={i} style={{display:"flex", gap:8, padding:"5px 4px", borderBottom: i<ACTIVITY.length-1 ? "1px solid var(--border-2)" : "none", alignItems:"flex-start"}}>
              <span className="dot" style={{background: sev, marginTop:6, flexShrink:0}}/>
              <div style={{flex:1, minWidth:0}}>
                <div style={{fontSize:11, lineHeight:1.4}}>{a.text}</div>
                <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", marginTop:1}}>{a.who} · {a.t}</div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

/* ---------------- Lane heatmap ---------------- */

function LaneHeatmap() {
  const regions = ["West", "Midwest", "South", "Northeast"];
  const grid = regions.map(o => regions.map(d => LANES.find(l => l.o === o && l.d === d)?.count || 0));
  const max = Math.max(...grid.flat());
  const total = grid.flat().reduce((a,b) => a+b, 0);
  return (
    <div className="card">
      <div style={{padding:"8px 12px", borderBottom:"1px solid var(--border)", display:"flex", justifyContent:"space-between", alignItems:"center"}}>
        <div style={{display:"flex", alignItems:"center", gap:8}}>
          <div className="label">Lane heatmap</div>
          <span className="mono" style={{fontSize:10, color:"var(--fg-subtle)"}}>origin → destination · {total} loads</span>
        </div>
        <button className="btn-ghost btn" style={{height:20, fontSize:10, padding:"0 6px"}}>7d</button>
      </div>
      <div style={{padding:12}}>
        <div style={{display:"grid", gridTemplateColumns:"60px repeat(4, 1fr)", gap:4, marginBottom:4}}>
          <div/>
          {regions.map(r => <div key={r} className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", textAlign:"center", textTransform:"uppercase", letterSpacing:"0.04em"}}>{r}</div>)}
        </div>
        {regions.map((o, i) => (
          <div key={o} style={{display:"grid", gridTemplateColumns:"60px repeat(4, 1fr)", gap:4, marginBottom:4}}>
            <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)", textTransform:"uppercase", letterSpacing:"0.04em", display:"flex", alignItems:"center"}}>{o}</div>
            {regions.map((d, j) => <HeatCell key={d} value={grid[i][j]} max={max}/>)}
          </div>
        ))}
        <div style={{marginTop:10, display:"flex", justifyContent:"space-between", alignItems:"center", fontSize:10, color:"var(--fg-subtle)"}}>
          <div className="mono">Top: West → Midwest (14)</div>
          <div style={{display:"flex", alignItems:"center", gap:4}}>
            <span className="mono" style={{fontSize:9}}>0</span>
            <div style={{width:80, height:6, borderRadius:3, background:"linear-gradient(to right, color-mix(in oklch, var(--brand) 5%, transparent), var(--brand))"}}/>
            <span className="mono" style={{fontSize:9}}>{max}</span>
          </div>
        </div>
      </div>
    </div>
  );
}

/* ---------------- Customer mix + tomorrow pickups (combined) ---------------- */

function CustomerMix() {
  const [tab, setTab] = useState("customers");
  return (
    <div className="card" style={{display:"flex", flexDirection:"column", height:280, minHeight:0, overflow:"hidden"}}>
      <div style={{padding:"6px 12px", borderBottom:"1px solid var(--border)", display:"flex", justifyContent:"space-between", alignItems:"center", flexShrink:0}}>
        <div style={{display:"flex", gap:2}}>
          {[["customers", "Customers"], ["pickups", "Tomorrow's pickups"]].map(([id, l]) => (
            <button key={id} onClick={() => setTab(id)} style={{
              border:"none", background:"transparent", padding:"5px 8px", borderRadius:4, cursor:"pointer",
              fontSize:11, fontWeight: tab === id ? 600 : 500,
              color: tab === id ? "var(--fg)" : "var(--fg-muted)",
              position:"relative"
            }}>
              {l}
              {tab === id && <span style={{position:"absolute", left:4, right:4, bottom:-7, height:2, background:"var(--brand)", borderRadius:2}}/>}
            </button>
          ))}
        </div>
        <button className="btn-ghost btn" style={{height:20, fontSize:10, padding:"0 6px"}}>30d</button>
      </div>
      {tab === "customers" ? (
        <div style={{padding:"10px 12px", display:"flex", flexDirection:"column", gap:6, overflowY:"auto", flex:1, minHeight:0}}>
          {CUSTOMERS.map((c, i) => (
            <div key={c.name} style={{display:"flex", alignItems:"center", gap:8, fontSize:11}}>
              <div style={{flex:1, minWidth:0}}>
                <div style={{display:"flex", justifyContent:"space-between", alignItems:"baseline"}}>
                  <span style={{fontWeight:500, whiteSpace:"nowrap", overflow:"hidden", textOverflow:"ellipsis"}}>{c.name}</span>
                  <span className="mono tabular" style={{color:"var(--fg-muted)", fontSize:10.5}}>${(c.revenue/1000).toFixed(1)}K · {c.loads}</span>
                </div>
                <div className="lane-bar" style={{marginTop:3}}>
                  <span style={{width: `${c.share * 2.5}%`, background: i === 0 ? "var(--brand)" : i === 1 ? "oklch(0.6 0.18 200)" : i === 2 ? "oklch(0.65 0.16 80)" : i === 3 ? "oklch(0.6 0.16 320)" : "var(--fg-muted)"}}/>
                </div>
              </div>
              <span className="mono tabular" style={{fontSize:10, color: c.trend > 0 ? "var(--success)" : c.trend < 0 ? "var(--danger)" : "var(--fg-subtle)", width:36, textAlign:"right"}}>
                {c.trend > 0 ? "▲" : c.trend < 0 ? "▼" : "–"}{Math.abs(c.trend)}%
              </span>
            </div>
          ))}
        </div>
      ) : (
        <div style={{display:"flex", flexDirection:"column", overflowY:"auto", flex:1, minHeight:0}}>
          {PICKUPS_TOMORROW.map((p, i) => (
            <div key={i} style={{padding:"6px 12px", borderBottom: i<PICKUPS_TOMORROW.length-1 ? "1px solid var(--border-2)" : "none", display:"flex", alignItems:"center", gap:8}}>
              <span className="mono tabular" style={{fontSize:11, fontWeight:600, width:42}}>{p.time}</span>
              <div style={{flex:1, minWidth:0}}>
                <div style={{fontSize:11, fontWeight:500}}>{p.customer}</div>
                <div className="mono" style={{fontSize:9.5, color:"var(--fg-subtle)"}}>{p.lane}</div>
              </div>
              {p.status === "unassigned"
                ? <span className="pill pill-soft-warning">Needs driver</span>
                : <span className="mono" style={{fontSize:10, color:"var(--fg-muted)"}}>{p.driver}</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function Footer() {
  return (
    <div style={{borderTop:"1px solid var(--border)", padding:"8px 16px", display:"flex", justifyContent:"space-between", alignItems:"center", fontSize:10, color:"var(--fg-subtle)", marginTop:"auto"}}>
      <div className="mono" style={{display:"flex", gap:14}}>
        <span><span className="dot pulse" style={{background:"var(--success)", color:"var(--success)", marginRight:4}}/>API · 142ms</span>
        <span>WS · connected</span>
        <span>EDI · 6 partners up</span>
        <span>Build 2026.4.20</span>
      </div>
      <div style={{display:"flex", gap:14}}>
        <a href="#" style={{color:"var(--fg-subtle)"}}>Help</a>
        <a href="#" style={{color:"var(--fg-subtle)"}}>Status</a>
        <a href="#" style={{color:"var(--fg-subtle)"}}>What's new</a>
      </div>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<App/>);
