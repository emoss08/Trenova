import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import type { LoadingOptimizationResult } from "@/types/loading-optimization";
import type { Shipment } from "@/types/shipment";
import { useFormContext, useWatch } from "react-hook-form";

export interface ShipmentMeta {
  shipmentId: string;
  proNumber: string;
  bol: string;
  customerName: string;
  originName: string;
  originAddress: string;
  destinationName: string;
  destinationAddress: string;
  trailerCode: string;
  driverName: string;
}

export function useShipmentMeta(): ShipmentMeta {
  const { control } = useFormContext<Shipment>();
  const shipmentId = useWatch({ control, name: "id" as any }) ?? "";
  const proNumber = useWatch({ control, name: "proNumber" as any }) ?? "";
  const bol = useWatch({ control, name: "bol" }) ?? "";
  const customer = useWatch({ control, name: "customer" as any });
  const moves = useWatch({ control, name: "moves" }) ?? [];

  const shipment = { moves } as Shipment;
  const pickupStop = getOriginStop(shipment);
  const deliveryStop = getDestinationStop(shipment);
  const assignment = moves[0]?.assignment;

  return {
    shipmentId: String(shipmentId || ""),
    proNumber: String(proNumber || ""),
    bol: String(bol || ""),
    customerName: customer?.name ?? "",
    originName: pickupStop?.location?.name ?? "",
    originAddress: formatStopAddress(pickupStop),
    destinationName: deliveryStop?.location?.name ?? "",
    destinationAddress: formatStopAddress(deliveryStop),
    trailerCode: assignment?.trailer?.code ?? "",
    driverName: assignment?.primaryWorker
      ? `${assignment.primaryWorker.firstName ?? ""} ${assignment.primaryWorker.lastName ?? ""}`.trim()
      : "",
  };
}

function formatStopAddress(stop: any): string {
  if (!stop?.location) return "";
  const loc = stop.location;
  const parts = [loc.addressLine1, loc.city, loc.state?.abbreviation].filter(Boolean);
  return parts.join(", ");
}

function formatDate() {
  return new Date().toLocaleString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function esc(s: string) {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function buildHandlingNotes(p: LoadingOptimizationResult["placements"][0]): string {
  const notes: string[] = [];
  if (p.isHazmat) notes.push(`<span style="color:#b45309;font-weight:700">\u2623 HAZMAT (${esc(p.hazmatClass ?? "")})</span>`);
  if (p.fragile) notes.push('<span style="color:#dc2626;font-weight:600">\u26a0 FRAGILE</span>');
  if (p.stackable) notes.push("Stackable");
  if (p.minTemp != null && p.maxTemp != null) notes.push(`\u2744 ${p.minTemp}\u2013${p.maxTemp}\u00b0F`);
  if (p.loadingInstructions) notes.push(esc(p.loadingInstructions));
  return notes.join(" &bull; ") || "\u2014";
}

export function buildLoadPlanBlob(data: LoadingOptimizationResult, meta: ShipmentMeta): Blob {
  return new Blob([buildLoadPlanHTML(data, meta)], { type: "text/html" });
}

export function printLoadPlan(data: LoadingOptimizationResult, meta: ShipmentMeta) {
  const w = window.open("", "_blank", "width=800,height=1100");
  if (!w) return;
  w.document.write(buildLoadPlanHTML(data, meta));
  w.document.close();
}

function buildLoadPlanHTML(data: LoadingOptimizationResult, meta: ShipmentMeta): string {
  const placements = data.placements;
  const axles = data.axleWeights.filter((a) => a.axle !== "steer");
  const allWarnings = data.warnings.filter((w) => w.severity !== "info");
  const recs = data.recommendations ?? [];

  const commodityRows = placements
    .map((p, i) => {
      const rowBg = i % 2 === 0 ? "" : 'style="background:#f9fafb"';
      return `<tr ${rowBg}>
        <td style="text-align:center;font-weight:700;color:#6b7280">${i + 1}</td>
        <td style="font-weight:600">${esc(p.commodityName)}</td>
        <td style="text-align:right;font-variant-numeric:tabular-nums">${p.weight.toLocaleString()}</td>
        <td style="text-align:right">${p.pieces}</td>
        <td style="text-align:right;font-variant-numeric:tabular-nums">${p.positionFeet}</td>
        <td style="text-align:right;font-variant-numeric:tabular-nums">${p.lengthFeet}${p.estimatedLength ? "*" : ""}</td>
        <td style="font-size:10px">${buildHandlingNotes(p)}</td>
      </tr>`;
    })
    .join("");

  const axleRows = axles
    .map(
      (a) => `<tr>
        <td style="text-transform:capitalize;font-weight:500">${a.axle} axle</td>
        <td style="text-align:right;font-variant-numeric:tabular-nums;${!a.compliant ? "color:#dc2626;font-weight:700" : ""}">${a.weight.toLocaleString()}</td>
        <td style="text-align:right;font-variant-numeric:tabular-nums;color:#6b7280">${a.limit.toLocaleString()}</td>
        <td style="text-align:center"><span style="display:inline-block;padding:1px 8px;border-radius:99px;font-size:10px;font-weight:600;${a.compliant ? "background:#dcfce7;color:#166534" : "background:#fee2e2;color:#991b1b"}">${a.compliant ? "PASS" : "FAIL"}</span></td>
      </tr>`,
    )
    .join("");

  const warningItems = allWarnings.map((warn) => {
    const color = warn.severity === "error" ? "#dc2626" : "#d97706";
    const icon = warn.severity === "error" ? "\u26d4" : "\u26a0";
    return `<div style="display:flex;gap:6px;align-items:flex-start;margin-bottom:4px"><span style="color:${color}">${icon}</span><span style="font-size:11px">${esc(warn.message)}</span></div>`;
  }).join("");

  const hasOriginDest = meta.originName || meta.destinationName;

  return `<!DOCTYPE html>
<html>
<head>
  <title>Load Plan \u2014 ${esc(meta.proNumber)}</title>
  <style>
    * { margin: 0; padding: 0; box-sizing: border-box; }
    body { font-family: -apple-system, "Segoe UI", system-ui, sans-serif; font-size: 12px; color: #111827; padding: 20px 28px; line-height: 1.4; }

    .header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 16px; padding-bottom: 12px; border-bottom: 2px solid #111827; }
    .header-left h1 { font-size: 20px; letter-spacing: -0.3px; margin-bottom: 2px; }
    .header-left .subtitle { font-size: 11px; color: #6b7280; }
    .header-right { text-align: right; font-size: 11px; color: #6b7280; }
    .header-right .pro { font-size: 16px; font-weight: 700; color: #111827; font-variant-numeric: tabular-nums; }

    .route-bar { display: flex; align-items: center; gap: 12px; background: #f3f4f6; border-radius: 6px; padding: 10px 14px; margin-bottom: 14px; }
    .route-point { flex: 1; }
    .route-point .label { font-size: 9px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px; color: #9ca3af; margin-bottom: 2px; }
    .route-point .name { font-weight: 600; font-size: 12px; }
    .route-point .addr { font-size: 10px; color: #6b7280; }
    .route-arrow { font-size: 18px; color: #d1d5db; flex-shrink: 0; }
    .route-info { border-left: 1px solid #d1d5db; padding-left: 12px; font-size: 10px; color: #6b7280; }
    .route-info strong { color: #374151; }

    .metrics { display: flex; gap: 1px; background: #e5e7eb; border-radius: 6px; overflow: hidden; margin-bottom: 14px; }
    .metric { flex: 1; background: white; padding: 8px 12px; text-align: center; }
    .metric .value { font-size: 18px; font-weight: 700; font-variant-numeric: tabular-nums; }
    .metric .label { font-size: 9px; text-transform: uppercase; letter-spacing: 0.5px; color: #6b7280; margin-top: 1px; }
    .metric.alert .value { color: #dc2626; }

    .section { margin-bottom: 14px; }
    .section-title { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.6px; color: #6b7280; margin-bottom: 6px; }

    table { width: 100%; border-collapse: collapse; }
    th { text-align: left; font-weight: 600; font-size: 9px; text-transform: uppercase; letter-spacing: 0.4px; color: #6b7280; padding: 6px 8px; border-bottom: 2px solid #e5e7eb; }
    td { padding: 6px 8px; border-bottom: 1px solid #f3f4f6; font-size: 11px; }

    .axle-table { width: auto; }
    .axle-table th, .axle-table td { padding: 5px 12px; }

    .warnings-box { background: #fffbeb; border: 1px solid #fde68a; border-radius: 6px; padding: 10px 12px; }

    .footer { margin-top: 20px; padding-top: 8px; border-top: 1px solid #e5e7eb; font-size: 9px; color: #9ca3af; display: flex; justify-content: space-between; }

    .two-col { display: flex; gap: 14px; }
    .two-col > :first-child { flex: 2; }
    .two-col > :last-child { flex: 1; }

    @media print {
      body { padding: 10px 16px; }
      .route-bar { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
      .metrics { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
    }
  </style>
</head>
<body>
  <div class="header">
    <div class="header-left">
      <h1>Load Plan</h1>
      <div class="subtitle">Trenova Transportation Management</div>
    </div>
    <div class="header-right">
      <div class="pro">${esc(meta.proNumber)}</div>
      <div>BOL: ${esc(meta.bol)}</div>
      ${meta.customerName ? `<div>${esc(meta.customerName)}</div>` : ""}
    </div>
  </div>

  ${hasOriginDest ? `
  <div class="route-bar">
    <div class="route-point">
      <div class="label">Origin</div>
      <div class="name">${esc(meta.originName)}</div>
      ${meta.originAddress ? `<div class="addr">${esc(meta.originAddress)}</div>` : ""}
    </div>
    <div class="route-arrow">\u2192</div>
    <div class="route-point">
      <div class="label">Destination</div>
      <div class="name">${esc(meta.destinationName)}</div>
      ${meta.destinationAddress ? `<div class="addr">${esc(meta.destinationAddress)}</div>` : ""}
    </div>
    ${meta.trailerCode || meta.driverName ? `
    <div class="route-info">
      ${meta.trailerCode ? `<div><strong>Trailer:</strong> ${esc(meta.trailerCode)}</div>` : ""}
      ${meta.driverName ? `<div><strong>Driver:</strong> ${esc(meta.driverName)}</div>` : ""}
    </div>` : ""}
  </div>` : ""}

  <div class="metrics">
    <div class="metric${data.linearFeetUtil > 100 ? " alert" : ""}">
      <div class="value">${data.totalLinearFeet.toFixed(1)}ft</div>
      <div class="label">of ${data.trailerLengthFeet}ft (${data.linearFeetUtil.toFixed(0)}%)</div>
    </div>
    <div class="metric${data.weightUtil > 100 ? " alert" : ""}">
      <div class="value">${data.totalWeight.toLocaleString()}</div>
      <div class="label">of ${data.maxWeight.toLocaleString()} lbs (${data.weightUtil.toFixed(0)}%)</div>
    </div>
    <div class="metric">
      <div class="value">${data.utilizationScore}%</div>
      <div class="label">${data.utilizationGrade} utilization</div>
    </div>
    <div class="metric">
      <div class="value">${placements.length}</div>
      <div class="label">Commodities</div>
    </div>
  </div>

  <div class="section">
    <div class="section-title">Loading Sequence (Load #1 first, near nose \u2192 last item near doors)</div>
    <table>
      <thead>
        <tr>
          <th style="width:32px;text-align:center">#</th>
          <th>Commodity</th>
          <th style="text-align:right">Weight (lbs)</th>
          <th style="text-align:right">Pcs</th>
          <th style="text-align:right">Position</th>
          <th style="text-align:right">Length</th>
          <th>Handling &amp; Notes</th>
        </tr>
      </thead>
      <tbody>${commodityRows}</tbody>
    </table>
  </div>

  ${recs.length > 0 ? `
  <div class="section">
    <div class="section-title">Recommendations</div>
    ${recs.map((r) => {
      const priorityLabel = r.priority === "critical" ? "\u26d4 CRITICAL" : r.priority === "suggested" ? "\u26a0 SUGGESTED" : "\u2139 TIP";
      const priorityStyle = r.priority === "critical"
        ? "color:#991b1b;background:#fee2e2;border:1px solid #fecaca"
        : r.priority === "suggested"
          ? "color:#92400e;background:#fef3c7;border:1px solid #fde68a"
          : "color:#1e40af;background:#dbeafe;border:1px solid #bfdbfe";
      return `<div style="display:flex;gap:8px;align-items:flex-start;margin-bottom:6px;padding:6px 10px;border-radius:4px;${priorityStyle}">
        <span style="font-size:9px;font-weight:700;white-space:nowrap;margin-top:1px">${priorityLabel}</span>
        <div style="font-size:11px">
          <strong>${esc(r.title)}</strong>
          <div style="margin-top:1px;opacity:0.8">${esc(r.description)}</div>
          ${r.impact ? `<div style="margin-top:2px;font-weight:600;font-size:10px">${esc(r.impact)}</div>` : ""}
        </div>
      </div>`;
    }).join("")}
  </div>` : ""}

  <div class="two-col">
    <div class="section">
      ${warningItems ? `
      <div class="section-title">Compliance Alerts</div>
      <div class="warnings-box">${warningItems}</div>
      ` : `
      <div style="display:flex;align-items:center;gap:6px;color:#166534;font-size:11px;margin-top:4px">
        <span style="font-size:14px">\u2705</span> No compliance issues detected
      </div>
      `}
    </div>
    <div class="section">
      <div class="section-title">Axle Weights</div>
      <table class="axle-table">
        <thead>
          <tr><th>Axle</th><th style="text-align:right">Weight</th><th style="text-align:right">Limit</th><th style="text-align:center">Status</th></tr>
        </thead>
        <tbody>${axleRows}</tbody>
      </table>
    </div>
  </div>

  ${placements.some((p) => p.estimatedLength) ? `
  <div style="font-size:9px;color:#9ca3af;margin-top:8px">* Length estimated from weight \u2014 configure linear feet per unit in commodity settings for accuracy.</div>
  ` : ""}

  <div class="footer">
    <span>Generated by Trenova Load Planner</span>
    <span>${formatDate()}</span>
  </div>

  <script>window.onload = function() { window.print(); }</script>
</body>
</html>`;
}
