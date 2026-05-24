/* Stylized US map with shipment routes & driver positions */

// US bounding box (approx contiguous): lon -125 to -66, lat 24 to 50
const LON_MIN = -125, LON_MAX = -66, LAT_MIN = 24, LAT_MAX = 50;

function project(lat, lon, w, h, pad = 12) {
  const x = pad + ((lon - LON_MIN) / (LON_MAX - LON_MIN)) * (w - pad * 2);
  const y = pad + (1 - (lat - LAT_MIN) / (LAT_MAX - LAT_MIN)) * (h - pad * 2);
  return [x, y];
}

// Approximated US silhouette (very simplified, decorative)
const US_PATH = "M 75 200 L 88 175 L 110 158 L 138 145 L 170 140 L 198 138 L 225 130 L 252 120 L 282 108 L 308 96 L 332 88 L 360 84 L 388 82 L 416 84 L 442 90 L 472 98 L 502 108 L 530 122 L 555 138 L 580 154 L 600 172 L 615 190 L 622 208 L 624 224 L 614 238 L 598 248 L 578 256 L 555 264 L 528 270 L 498 273 L 466 274 L 438 270 L 412 264 L 388 256 L 360 246 L 330 240 L 298 240 L 270 246 L 244 254 L 220 264 L 195 272 L 170 278 L 144 280 L 122 274 L 102 262 L 88 246 L 78 226 Z";

// Approximate state cluster blobs (decorative)
const STATE_BLOBS = [
  // [cx, cy, rx, ry] in 720x320 viewBox
  [120, 180, 32, 28], [165, 195, 28, 26], [205, 200, 30, 24], [250, 195, 30, 24],
  [300, 195, 32, 24], [350, 195, 30, 24], [400, 195, 30, 24], [450, 195, 28, 24],
  [495, 200, 30, 26], [540, 215, 28, 24], [575, 230, 26, 22],
  [165, 155, 28, 22], [210, 152, 28, 20], [255, 150, 28, 22], [305, 148, 32, 22], [355, 150, 32, 22], [405, 150, 30, 22], [450, 152, 28, 22], [495, 158, 30, 22], [535, 168, 26, 22],
  [180, 122, 30, 22], [225, 118, 32, 22], [275, 114, 32, 22], [325, 110, 32, 22], [375, 108, 32, 22], [425, 110, 32, 22], [475, 116, 30, 22], [515, 126, 28, 22],
  [220, 100, 28, 20], [275, 92, 32, 20], [325, 88, 30, 20], [380, 88, 30, 20], [435, 92, 28, 20],
  [305, 230, 32, 24], [355, 235, 32, 24], [405, 235, 32, 24], [455, 230, 28, 22],
  [240, 240, 30, 22], [290, 250, 30, 22], [345, 258, 30, 22], [395, 258, 28, 22], [445, 252, 26, 22],
];

function ShipmentMap({ shipments, drivers, mapStyle = "tactical", highlightId, onHover, onSelect }) {
  const W = 720, H = 320;
  const wrapRef = React.useRef(null);
  const [tick, setTick] = React.useState(0);
  React.useEffect(() => {
    const id = setInterval(() => setTick(t => t + 1), 1500);
    return () => clearInterval(id);
  }, []);

  const styleConf = {
    tactical: { land: "var(--map-land)", landStroke: "var(--border)", water: "var(--map-bg)", grid: "var(--border-2)", route: "var(--brand)" },
    street:   { land: "oklch(0.93 0.01 90)", landStroke: "oklch(0.85 0.01 90)", water: "oklch(0.88 0.03 230)", grid: "oklch(0.83 0.01 90)", route: "var(--brand)" },
    satellite:{ land: "oklch(0.22 0.05 145)", landStroke: "oklch(0.30 0.04 145)", water: "oklch(0.18 0.05 240)", grid: "oklch(0.30 0.04 145)", route: "oklch(0.85 0.16 86)" },
  }[mapStyle] || {};

  // City coordinate lookup so each shipment uses its real origin & destination
  const CITY = {
    "Long Beach, CA":      [33.77, -118.19],
    "Los Angeles, CA":     [34.05, -118.24],
    "Phoenix, AZ":         [33.45, -112.07],
    "Cleveland, OH":       [41.50, -81.69],
    "Atlanta, GA":         [33.75, -84.39],
    "Denver, CO":          [39.74, -104.99],
    "Seattle, WA":         [47.61, -122.33],
    "Houston, TX":         [29.76, -95.37],
    "Chicago, IL":         [41.85, -87.65],
    "Customer DC, TX":     [32.78, -96.80],
    "Detroit, MI":         [42.33, -83.05],
    "Charlotte, NC":       [35.23, -80.84],
    "Salt Lake City, UT":  [40.76, -111.89],
    "Portland, OR":        [45.52, -122.68],
    "New Orleans, LA":     [29.95, -90.07],
  };

  // Generate routes from shipments with coords
  const routes = shipments.filter(s => s.lat && s.lon).map(s => {
    const o = CITY[s.origin] || [34, -118];
    const d = CITY[s.dest] || [41.85, -87.65];
    const [ox, oy] = project(o[0], o[1], W, H);
    const [tx, ty] = project(s.lat, s.lon, W, H);
    const [dx, dy] = project(d[0], d[1], W, H);
    return { id: s.id, status: s.status, ox, oy, tx, ty, dx, dy, progress: s.progress };
  });

  return (
    <div ref={wrapRef} style={{ position: "relative", width: "100%", height: "100%", background: styleConf.water, overflow: "hidden", borderRadius: 6 }}>
      {/* grid backdrop */}
      <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="xMidYMid slice" style={{ position:"absolute", inset:0, width:"100%", height:"100%" }}>
        <defs>
          <pattern id="mapgrid" width="24" height="24" patternUnits="userSpaceOnUse">
            <path d="M 24 0 L 0 0 0 24" fill="none" stroke={styleConf.grid} strokeWidth="0.4" opacity="0.4"/>
          </pattern>
          <radialGradient id="vignette" cx="50%" cy="50%" r="60%">
            <stop offset="60%" stopColor="black" stopOpacity="0"/>
            <stop offset="100%" stopColor="black" stopOpacity="0.35"/>
          </radialGradient>
        </defs>
        <rect width={W} height={H} fill="url(#mapgrid)"/>

        {/* state blobs */}
        <g>
          {STATE_BLOBS.map(([cx, cy, rx, ry], i) => (
            <ellipse key={i} cx={cx} cy={cy} rx={rx} ry={ry}
                     fill={styleConf.land} stroke={styleConf.landStroke} strokeWidth="0.6" opacity="0.95"/>
          ))}
        </g>

        {/* lane labels — sparingly */}
        <g fontFamily="Geist Mono, monospace" fontSize="8" fill="var(--fg-subtle)" opacity="0.6">
          <text x={130} y={205}>CA</text>
          <text x={250} y={185}>UT</text>
          <text x={350} y={210}>TX</text>
          <text x={420} y={235}>LA</text>
          <text x={460} y={195}>TN</text>
          <text x={510} y={170}>OH</text>
          <text x={555} y={155}>NY</text>
          <text x={310} y={130}>SD</text>
          <text x={400} y={150}>IL</text>
        </g>

        {/* Routes */}
        <g>
          {routes.map((r, i) => {
            const dasharray = r.status === "Unassigned" ? "3 3" : "0";
            const isHigh = highlightId === r.id;
            const opacity = highlightId && !isHigh ? 0.25 : 0.95;
            const color = r.status === "At Risk" ? "var(--danger)"
                        : r.status === "Detention" ? "var(--warning)"
                        : r.status === "Delivered" ? "var(--success)"
                        : styleConf.route;
            // Curved path from origin → current → destination
            const cx = (r.ox + r.dx) / 2, cy = Math.min(r.oy, r.dy) - 30;
            return (
              <g key={r.id} opacity={opacity}>
                {/* completed portion */}
                <path d={`M ${r.ox} ${r.oy} Q ${(r.ox+r.tx)/2} ${(r.oy+r.ty)/2 - 18} ${r.tx} ${r.ty}`}
                      fill="none" stroke={color} strokeWidth={isHigh ? 2 : 1.4} strokeLinecap="round"/>
                {/* remaining (dashed) */}
                <path d={`M ${r.tx} ${r.ty} Q ${(r.tx+r.dx)/2} ${(r.ty+r.dy)/2 - 12} ${r.dx} ${r.dy}`}
                      fill="none" stroke={color} strokeWidth={isHigh ? 1.6 : 1} strokeDasharray="3 3" opacity="0.55"/>
              </g>
            );
          })}
        </g>

        {/* origin/dest pins */}
        <g>
          {routes.map((r) => (
            <g key={r.id + "-pins"}>
              <circle cx={r.ox} cy={r.oy} r="2.5" fill={styleConf.water} stroke="var(--fg-muted)" strokeWidth="1"/>
              <circle cx={r.dx} cy={r.dy} r="2.5" fill={styleConf.water} stroke="var(--fg-muted)" strokeWidth="1"/>
            </g>
          ))}
        </g>

        {/* current driver positions (animated pulse) */}
        <g>
          {routes.map((r) => {
            const isHigh = highlightId === r.id;
            const color = r.status === "At Risk" ? "var(--danger)"
                        : r.status === "Detention" ? "var(--warning)"
                        : r.status === "Delivered" ? "var(--success)"
                        : "var(--brand)";
            return (
              <g key={r.id + "-truck"} style={{cursor:"pointer"}} onMouseEnter={() => onHover && onHover(r.id)} onMouseLeave={() => onHover && onHover(null)} onClick={() => onSelect && onSelect(r.id)}>
                <circle cx={r.tx} cy={r.ty} r={isHigh ? 9 : 6} fill={color} opacity="0.18">
                  <animate attributeName="r" values={`${isHigh?9:6};${isHigh?14:11};${isHigh?9:6}`} dur="2.2s" repeatCount="indefinite"/>
                  <animate attributeName="opacity" values="0.32;0;0.32" dur="2.2s" repeatCount="indefinite"/>
                </circle>
                <circle cx={r.tx} cy={r.ty} r={isHigh ? 4 : 3} fill={color} stroke="var(--card)" strokeWidth="1"/>
              </g>
            );
          })}
        </g>

        <rect width={W} height={H} fill="url(#vignette)" pointerEvents="none"/>
      </svg>

      {/* Map chrome overlays */}
      <div style={{ position: "absolute", top: 8, left: 8, display: "flex", gap: 6 }}>
        <div className="card mono" style={{ padding: "4px 8px", fontSize: 10, display: "flex", alignItems: "center", gap: 6, background: "color-mix(in oklch, var(--card) 80%, transparent)", backdropFilter: "blur(4px)" }}>
          <span className="dot pulse" style={{ background: "var(--success)", color: "var(--success)" }}/>
          LIVE · {drivers.length} units
        </div>
        <div className="card mono" style={{ padding: "4px 8px", fontSize: 10, color: "var(--fg-muted)", background: "color-mix(in oklch, var(--card) 80%, transparent)", backdropFilter: "blur(4px)" }}>
          synced {tick * 1.5 < 60 ? `${Math.floor(tick * 1.5)}s` : `${Math.floor(tick * 1.5 / 60)}m`} ago
        </div>
      </div>

      {/* Legend */}
      <div style={{ position: "absolute", bottom: 8, left: 8, display: "flex", gap: 6, alignItems: "center", flexWrap: "wrap", maxWidth: "calc(100% - 60px)" }}>
        {[
          ["In transit", "var(--brand)"],
          ["At risk", "var(--danger)"],
          ["Detention", "var(--warning)"],
          ["Delivered", "var(--success)"],
        ].map(([l, c]) => (
          <div key={l} className="mono" style={{ fontSize: 10, color: "var(--fg-muted)", display: "flex", alignItems: "center", gap: 4, padding: "2px 6px", background: "color-mix(in oklch, var(--card) 80%, transparent)", border:"1px solid var(--border)", borderRadius: 3, backdropFilter: "blur(4px)", whiteSpace:"nowrap" }}>
            <span className="dot" style={{ background: c }}/>{l}
          </div>
        ))}
      </div>

      {/* Map controls */}
      <div style={{ position: "absolute", top: 8, right: 8, display: "flex", flexDirection: "column", gap: 4 }}>
        {[<IcLayers size={12}/>, <IcMap size={12}/>, <IcRadar size={12}/>, "+", "−"].map((c, i) => (
          <button key={i} className="btn" style={{ width: 24, height: 24, padding: 0, justifyContent: "center", background: "color-mix(in oklch, var(--card) 80%, transparent)", backdropFilter: "blur(4px)" }}>{c}</button>
        ))}
      </div>
    </div>
  );
}

Object.assign(window, { ShipmentMap });
