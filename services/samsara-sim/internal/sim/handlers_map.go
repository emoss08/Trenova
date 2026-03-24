package sim

import (
	"net/http"
)

func (s *Server) handleMapView(writer http.ResponseWriter, request *http.Request) {
	_ = request
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	if _, err := writer.Write([]byte(simMapHTML)); err != nil {
		s.logger.Error("failed to render simulator map", "error", err.Error())
	}
}

const simMapHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Samsara Sim Live Map</title>
  <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" integrity="sha256-p4NxAoJBhIIN+hmNHrzRCf9tD/miZyoHS5obTRR9BMY=" crossorigin="" />
  <style>
    :root {
      --bg: #0f172a;
      --panel: #111827;
      --panel2: #1f2937;
      --text: #e5e7eb;
      --muted: #94a3b8;
      --ok: #10b981;
      --warn: #f59e0b;
      --accent: #3b82f6;
      --border: #334155;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: ui-sans-serif, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif;
      color: var(--text);
      background: radial-gradient(1200px 800px at 20% 0%, #1e293b 0%, var(--bg) 55%);
      min-height: 100vh;
    }
    .layout {
      display: grid;
      grid-template-rows: auto 1fr;
      min-height: 100vh;
    }
    .toolbar {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      align-items: center;
      padding: 12px 14px;
      border-bottom: 1px solid var(--border);
      background: rgba(15, 23, 42, 0.9);
      backdrop-filter: blur(8px);
      position: sticky;
      top: 0;
      z-index: 500;
    }
    .toolbar label {
      font-size: 12px;
      color: var(--muted);
      display: flex;
      flex-direction: column;
      gap: 4px;
      min-width: 120px;
    }
    .toolbar input, .toolbar button {
      height: 34px;
      border-radius: 8px;
      border: 1px solid var(--border);
      background: var(--panel2);
      color: var(--text);
      padding: 0 10px;
      font-size: 13px;
    }
    .toolbar button {
      background: linear-gradient(180deg, #2563eb, #1d4ed8);
      border: 0;
      cursor: pointer;
      font-weight: 600;
      padding: 0 14px;
    }
    .toolbar button.secondary {
      background: var(--panel2);
      border: 1px solid var(--border);
    }
    .status {
      font-size: 12px;
      color: var(--muted);
      margin-left: auto;
      max-width: 520px;
      text-align: right;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .chips {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
      align-items: center;
    }
    .chip {
      height: 28px;
      border-radius: 999px;
      border: 1px solid var(--border);
      background: var(--panel2);
      color: var(--muted);
      padding: 0 10px;
      font-size: 11px;
      cursor: pointer;
    }
    .chip.active {
      background: rgba(59, 130, 246, 0.18);
      border-color: #3b82f6;
      color: #bfdbfe;
    }
    .content {
      display: grid;
      grid-template-columns: minmax(0, 1fr) 340px;
      min-height: 0;
    }
    #map {
      min-height: 520px;
      height: calc(100vh - 60px);
    }
    .sidebar {
      border-left: 1px solid var(--border);
      background: rgba(17, 24, 39, 0.92);
      display: flex;
      flex-direction: column;
      min-height: 0;
      overflow: hidden;
    }
    .panel-section {
      display: flex;
      flex-direction: column;
      min-height: 0;
      flex: 1 1 0;
      border-bottom: 1px solid var(--border);
    }
    .panel-section:last-child {
      border-bottom: 0;
    }
    .section-title {
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 0.06em;
      color: var(--muted);
      padding: 10px 12px 8px 12px;
      border-bottom: 1px solid var(--border);
    }
    .list {
      overflow: auto;
      padding: 10px 10px 14px 10px;
      min-height: 0;
      display: flex;
      flex-direction: column;
      gap: 8px;
    }
    .card {
      border: 1px solid var(--border);
      background: rgba(31, 41, 55, 0.9);
      border-radius: 10px;
      padding: 8px 10px;
      font-size: 12px;
      line-height: 1.4;
    }
    .card.interactive {
      cursor: pointer;
      transition: border-color 140ms ease, box-shadow 140ms ease;
    }
    .card.interactive:hover {
      border-color: #64748b;
      box-shadow: 0 0 0 1px rgba(100, 116, 139, 0.35);
    }
    .card.selected {
      border-color: #60a5fa;
      box-shadow: 0 0 0 1px rgba(96, 165, 250, 0.45);
    }
    .card .title {
      display: flex;
      justify-content: space-between;
      gap: 8px;
      margin-bottom: 4px;
      font-size: 13px;
      font-weight: 700;
    }
    .pill {
      font-size: 11px;
      border-radius: 999px;
      padding: 1px 7px;
      border: 1px solid var(--border);
      color: var(--muted);
    }
    .pill.ok { color: var(--ok); border-color: #047857; }
    .pill.warn { color: var(--warn); border-color: #92400e; }
    .pill.info { color: #93c5fd; border-color: #1d4ed8; }
    .pill.bad { color: #fca5a5; border-color: #b91c1c; }
    .event-badges {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin: 0 0 6px 0;
    }
    .event-pill {
      font-size: 10px;
      border-radius: 999px;
      border: 1px solid var(--border);
      padding: 1px 6px;
      color: var(--muted);
      background: rgba(15, 23, 42, 0.6);
    }
    .event-pill.speeding {
      color: #fca5a5;
      border-color: #b91c1c;
    }
    .event-pill.violation {
      color: #fdba74;
      border-color: #9a3412;
    }
    .event-pill.stop {
      color: #93c5fd;
      border-color: #1d4ed8;
    }
    .event-pill.duty {
      color: #86efac;
      border-color: #166534;
    }
    .kv {
      display: grid;
      grid-template-columns: 88px 1fr;
      gap: 4px 8px;
      color: var(--muted);
    }
    .kv b { color: var(--text); font-weight: 600; }
    .stop-chips {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;
      margin: 6px 0 0 0;
    }
    .stop-chip {
      border: 1px solid var(--border);
      border-radius: 999px;
      background: rgba(15, 23, 42, 0.8);
      color: var(--muted);
      font-size: 10px;
      padding: 2px 8px;
      cursor: pointer;
    }
    .stop-chip.selected {
      border-color: #60a5fa;
      color: #bfdbfe;
      background: rgba(59, 130, 246, 0.18);
    }
    @media (max-width: 1000px) {
      .content { grid-template-columns: 1fr; }
      .sidebar {
        border-left: 0;
        border-top: 1px solid var(--border);
        min-height: 520px;
      }
      #map { height: 58vh; min-height: 380px; }
      .status { width: 100%; margin-left: 0; text-align: left; }
    }
  </style>
</head>
<body>
  <div class="layout">
    <div class="toolbar">
      <label>Bearer Token
        <input id="token" value="dev-samsara-token" />
      </label>
      <label>Poll (ms)
        <input id="interval" type="number" min="500" step="250" value="1000" />
      </label>
      <button id="apply">Apply</button>
      <button class="secondary" id="pause">Pause</button>
      <button class="secondary" id="focus">Focus Fleet</button>
      <button class="secondary" id="clear-webhooks">Clear Inbox</button>
      <div class="chips" id="route-filters">
        <button class="chip active" type="button" data-route-filter="all">all</button>
        <button class="chip" type="button" data-route-filter="planned">planned</button>
        <button class="chip" type="button" data-route-filter="assigned">assigned</button>
        <button class="chip" type="button" data-route-filter="enRoute">en route</button>
        <button class="chip" type="button" data-route-filter="atStop">at stop</button>
        <button class="chip" type="button" data-route-filter="completed">completed</button>
        <button class="chip" type="button" data-route-filter="canceled">canceled</button>
      </div>
      <div class="status" id="status">starting...</div>
    </div>

    <div class="content">
      <div id="map"></div>
      <div class="sidebar">
        <section class="panel-section">
          <div class="section-title">Assets</div>
          <div class="list" id="asset-list"></div>
        </section>
        <section class="panel-section">
          <div class="section-title">Driver HOS</div>
          <div class="list" id="hos-list"></div>
        </section>
        <section class="panel-section">
          <div class="section-title">Webhook Inbox</div>
          <div class="list" id="webhook-list"></div>
        </section>
      </div>
    </div>
  </div>

  <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js" integrity="sha256-20nQCchB9co0qIjJZRGuk2/Z9VM+kNiyxNV1lvTlZBo=" crossorigin=""></script>
  <script>
    (function () {
      const state = {
        token: "dev-samsara-token",
        pollMs: 1000,
        running: true,
        busy: false,
        timer: null,
        assetIDs: [],
        markers: new Map(),
        vehicleStats: new Map(),
        routeLifecycleByVehicle: new Map(),
        activeEventsByVehicle: new Map(),
        activeEventsByDriver: new Map(),
        routePaths: new Map(),
        stopMarkers: new Map(),
        assetTrailPoints: new Map(),
        routeStatusFilter: "all",
        selectedAssetID: "",
        selectedStopID: "",
        followSelected: false,
        selectedTrailLine: null,
        selectedTrailAssetID: "",
        webhookRecords: [],
        routesLoaded: false,
        fitDone: false
      };

      const elToken = document.getElementById("token");
      const elInterval = document.getElementById("interval");
      const elStatus = document.getElementById("status");
      const elAssetList = document.getElementById("asset-list");
      const elHOSList = document.getElementById("hos-list");
      const elWebhookList = document.getElementById("webhook-list");
      const elRouteFilters = document.getElementById("route-filters");
      const btnApply = document.getElementById("apply");
      const btnPause = document.getElementById("pause");
      const btnFocus = document.getElementById("focus");
      const btnClearWebhooks = document.getElementById("clear-webhooks");

      const routeStatusFilters = [
        "all",
        "planned",
        "assigned",
        "enRoute",
        "atStop",
        "completed",
        "canceled"
      ];

      const map = L.map("map", { zoomControl: true }).setView([31.4, -97.5], 6);
      L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
        maxZoom: 19,
        attribution: "&copy; OpenStreetMap contributors"
      }).addTo(map);

      function setStatus(text) {
        elStatus.textContent = text;
      }

      function hashColor(id) {
        let hash = 0;
        for (let i = 0; i < id.length; i++) {
          hash = (hash << 5) - hash + id.charCodeAt(i);
          hash |= 0;
        }
        const hue = Math.abs(hash) % 360;
        return "hsl(" + hue + " 80% 55%)";
      }

      function rememberTrailPoint(assetID, lat, lon) {
        const cleanAssetID = String(assetID || "").trim();
        if (!cleanAssetID) return;
        if (!Number.isFinite(lat) || !Number.isFinite(lon)) return;
        if (!state.assetTrailPoints.has(cleanAssetID)) {
          state.assetTrailPoints.set(cleanAssetID, []);
        }
        const points = state.assetTrailPoints.get(cleanAssetID);
        const last = points.length > 0 ? points[points.length - 1] : null;
        if (last && Math.abs(last[0] - lat) < 0.000001 && Math.abs(last[1] - lon) < 0.000001) {
          return;
        }
        points.push([lat, lon]);
        const maxPoints = 900;
        if (points.length > maxPoints) {
          points.splice(0, points.length - maxPoints);
        }
      }

      function updateSelectedTrail() {
        if (!state.selectedAssetID) {
          if (state.selectedTrailLine) {
            state.selectedTrailLine.remove();
          }
          state.selectedTrailLine = null;
          state.selectedTrailAssetID = "";
          return;
        }

        const points = state.assetTrailPoints.get(state.selectedAssetID) || [];
        if (points.length < 2) {
          if (state.selectedTrailLine) {
            state.selectedTrailLine.remove();
            state.selectedTrailLine = null;
            state.selectedTrailAssetID = "";
          }
          return;
        }

        const color = hashColor(state.selectedAssetID);
        if (!state.selectedTrailLine || state.selectedTrailAssetID !== state.selectedAssetID) {
          if (state.selectedTrailLine) {
            state.selectedTrailLine.remove();
          }
          state.selectedTrailLine = L.polyline(points, {
            color: color,
            weight: 4,
            opacity: 0.9
          }).addTo(map);
          state.selectedTrailAssetID = state.selectedAssetID;
          return;
        }

        state.selectedTrailLine.setLatLngs(points);
        state.selectedTrailLine.setStyle({ color: color });
      }

      function focusAsset(assetID, sourceLabel) {
        const cleanAssetID = String(assetID || "").trim();
        if (!cleanAssetID) return;
        const selectedRoute = state.routeLifecycleByVehicle.get(cleanAssetID) || {};
        const selectedStatus = normalizeRouteStatus(selectedRoute.status);
        if (!isRouteVisibleForFilter(selectedStatus)) {
          state.routeStatusFilter = "all";
          updateRouteFilterChips();
        }
        const marker = state.markers.get(cleanAssetID);
        if (!marker) {
          setStatus("asset " + cleanAssetID + " is not visible yet");
          return;
        }
        if (state.selectedAssetID !== cleanAssetID) {
          state.selectedStopID = "";
        }
        state.selectedAssetID = cleanAssetID;
        state.followSelected = true;
        if (!state.selectedStopID) {
          const currentStop = String(((selectedRoute.progress || {}).currentStopId || "")).trim();
          const nextStop = String(((selectedRoute.progress || {}).nextStopId || "")).trim();
          state.selectedStopID = currentStop || nextStop;
        }

        const latLng = marker.getLatLng();
        const zoom = Math.max(map.getZoom(), 13);
        map.setView([latLng.lat, latLng.lng], zoom, { animate: true });
        marker.openPopup();
        applyRouteVisibilityToPaths();
        updateSelectedTrail();
        renderSelectedRouteStops();
        renderAssets(Array.from(state.vehicleStats.values()));

        const via = sourceLabel ? " via " + sourceLabel : "";
        setStatus("tracking " + cleanAssetID + via);
      }

      function bindAssetCardClicks() {
        const cards = elAssetList.querySelectorAll(".asset-card[data-asset-id]");
        cards.forEach((card) => {
          card.addEventListener("click", function () {
            const assetID = String(card.getAttribute("data-asset-id") || "").trim();
            focusAsset(assetID, "asset");
          });
        });
      }

      function bindHOSCardClicks() {
        const cards = elHOSList.querySelectorAll(".hos-card[data-driver-id]");
        cards.forEach((card) => {
          card.addEventListener("click", function () {
            const vehicleID = String(card.getAttribute("data-vehicle-id") || "").trim();
            const driverID = String(card.getAttribute("data-driver-id") || "").trim();
            if (!vehicleID) {
              setStatus("driver " + driverID + " has no assigned vehicle");
              return;
            }
            focusAsset(vehicleID, "driver");
          });
        });
      }

      function formatMS(ms) {
        const value = Number(ms || 0);
        const totalSeconds = Math.max(0, Math.floor(value / 1000));
        const hours = Math.floor(totalSeconds / 3600);
        const mins = Math.floor((totalSeconds % 3600) / 60);
        const secs = totalSeconds % 60;
        return String(hours).padStart(2, "0") + ":" +
          String(mins).padStart(2, "0") + ":" +
          String(secs).padStart(2, "0");
      }

      function classifyEventType(eventType) {
        if (String(eventType).startsWith("speeding.")) return "speeding";
        if (String(eventType).startsWith("hos.violation.")) return "violation";
        if (String(eventType).startsWith("stop.")) return "stop";
        if (String(eventType).startsWith("duty.")) return "duty";
        return "generic";
      }

      function eventLabel(eventType) {
        const value = String(eventType || "").trim();
        if (value === "speeding.burst_major") return "speed major";
        if (value === "speeding.burst_minor") return "speed minor";
        if (value === "stop.traffic_delay") return "traffic";
        if (value === "stop.fuel_break") return "fuel stop";
        if (value === "duty.off_duty_pause") return "off duty";
        if (value === "duty.sleeper_berth_block") return "sleeper";
        if (value === "hos.violation.break") return "break vio";
        if (value === "hos.violation.drive") return "drive vio";
        if (value === "hos.violation.shift") return "shift vio";
        if (value === "hos.violation.cycle") return "cycle vio";
        return value.replaceAll(".", " ");
      }

      function escapeHTML(value) {
        return String(value || "")
          .replaceAll("&", "&amp;")
          .replaceAll("<", "&lt;")
          .replaceAll(">", "&gt;")
          .replaceAll('"', "&quot;")
          .replaceAll("'", "&#39;");
      }

      function renderEventBadges(events) {
        if (!Array.isArray(events) || events.length === 0) return "";
        const pills = [];
        for (const event of events.slice(0, 3)) {
          const eventType = String(event.type || "");
          const cls = classifyEventType(eventType);
          pills.push(
            "<span class='event-pill " + cls + "'>" + eventLabel(eventType) + "</span>"
          );
        }
        return "<div class='event-badges'>" + pills.join("") + "</div>";
      }

      function normalizeRouteStatus(status) {
        const value = String(status || "").trim();
        if (value === "") return "planned";
        for (const filterValue of routeStatusFilters) {
          if (filterValue !== "all" && filterValue === value) {
            return value;
          }
        }
        return value;
      }

      function isRouteVisibleForFilter(status) {
        if (state.routeStatusFilter === "all") {
          return true;
        }
        return normalizeRouteStatus(status) === state.routeStatusFilter;
      }

      function routeStatusText(status) {
        const value = String(status || "").trim();
        if (value === "all") return "all";
        if (value === "enRoute") return "en route";
        if (value === "atStop") return "at stop";
        if (value === "completed") return "completed";
        if (value === "assigned") return "assigned";
        if (value === "planned") return "planned";
        if (value === "canceled") return "canceled";
        if (value === "pending") return "pending";
        if (value === "missed") return "missed";
        return value || "unassigned";
      }

      function routeStatusClass(status) {
        const value = String(status || "").trim();
        if (value === "completed") return "ok";
        if (value === "atStop" || value === "enRoute") return "info";
        if (value === "canceled") return "bad";
        return "warn";
      }

      function updateRouteFilterChips() {
        if (!elRouteFilters) return;
        const counts = new Map();
        for (const value of routeStatusFilters) {
          counts.set(value, 0);
        }
        for (const assetID of state.assetIDs) {
          const route = state.routeLifecycleByVehicle.get(assetID) || {};
          const status = normalizeRouteStatus(route.status);
          counts.set("all", Number(counts.get("all") || 0) + 1);
          counts.set(status, Number(counts.get(status) || 0) + 1);
        }

        const chips = elRouteFilters.querySelectorAll(".chip[data-route-filter]");
        chips.forEach((chip) => {
          const value = String(chip.getAttribute("data-route-filter") || "").trim();
          chip.classList.toggle("active", value === state.routeStatusFilter);
          const count = Number(counts.get(value) || 0);
          chip.textContent = routeStatusText(value) + " (" + count + ")";
        });
      }

      function applyRouteVisibilityToPaths() {
        for (const [assetID, line] of state.routePaths.entries()) {
          const route = state.routeLifecycleByVehicle.get(assetID) || {};
          const status = normalizeRouteStatus(route.status);
          const isVisible = isRouteVisibleForFilter(status);
          if (!isVisible) {
            line.remove();
            continue;
          }
          if (!map.hasLayer(line)) {
            line.addTo(map);
          }
          const isSelected = state.selectedAssetID === assetID;
          line.setStyle({
            color: hashColor(assetID),
            weight: isSelected ? 5 : 3,
            opacity: isSelected ? 0.88 : 0.55,
            dashArray: isSelected ? "" : "5,7"
          });
        }
      }

      function stopStatusColor(status) {
        const value = String(status || "").trim();
        if (value === "completed") return "#22c55e";
        if (value === "atStop") return "#f59e0b";
        if (value === "missed") return "#ef4444";
        return "#94a3b8";
      }

      function clearStopMarkers() {
        for (const entry of state.stopMarkers.values()) {
          entry.marker.remove();
        }
        state.stopMarkers.clear();
      }

      function highlightSelectedStopMarker(shouldFocus) {
        let selectedMarker = null;
        for (const [stopID, entry] of state.stopMarkers.entries()) {
          const isSelected = state.selectedStopID !== "" && state.selectedStopID === stopID;
          const baseColor = stopStatusColor(entry.status);
          const markerColor = isSelected ? "#60a5fa" : baseColor;
          entry.marker.setStyle({
            radius: isSelected ? 7 : 5,
            color: markerColor,
            fillColor: markerColor,
            fillOpacity: isSelected ? 1 : 0.9,
            weight: isSelected ? 3 : 2
          });
          if (isSelected) {
            selectedMarker = entry.marker;
          }
        }

        if (shouldFocus && selectedMarker) {
          const latLng = selectedMarker.getLatLng();
          map.panTo([latLng.lat, latLng.lng], { animate: true, duration: 0.3 });
          selectedMarker.openPopup();
        }
      }

      function focusStop(assetID, stopID, sourceLabel) {
        const cleanAssetID = String(assetID || "").trim();
        const cleanStopID = String(stopID || "").trim();
        if (!cleanStopID) return;
        if (cleanAssetID && state.selectedAssetID !== cleanAssetID) {
          focusAsset(cleanAssetID, sourceLabel || "stop");
        }
        state.selectedStopID = cleanStopID;
        highlightSelectedStopMarker(true);
        renderAssets(Array.from(state.vehicleStats.values()));
      }

      function renderStopChips(assetID, route) {
        const stops = Array.isArray((route || {}).stops) ? route.stops : [];
        if (stops.length === 0) return "";
        const chips = [];
        const cleanAssetID = escapeHTML(assetID);
        for (const stop of stops) {
          const stopID = String(stop.id || "").trim();
          if (!stopID) continue;
          const label = escapeHTML(String(stop.name || stopID));
          const cleanStopID = escapeHTML(stopID);
          const isSelected =
            state.selectedAssetID === assetID && state.selectedStopID === stopID;
          const stopStatus = String(stop.status || "pending").trim();
          chips.push(
            "<button type='button' class='stop-chip" + (isSelected ? " selected" : "") +
              "' data-asset-id='" + cleanAssetID + "' data-stop-id='" + cleanStopID +
              "' title='" + escapeHTML(routeStatusText(stopStatus)) + "'>" + label + "</button>"
          );
        }
        if (chips.length === 0) return "";
        return "<div class='stop-chips'>" + chips.join("") + "</div>";
      }

      function bindStopChipClicks() {
        const chips = elAssetList.querySelectorAll(".stop-chip[data-stop-id]");
        chips.forEach((chip) => {
          chip.addEventListener("click", function (event) {
            event.stopPropagation();
            const assetID = String(chip.getAttribute("data-asset-id") || "").trim();
            const stopID = String(chip.getAttribute("data-stop-id") || "").trim();
            focusStop(assetID, stopID, "stop");
          });
        });
      }

      function renderSelectedRouteStops() {
        clearStopMarkers();
        const assetID = String(state.selectedAssetID || "").trim();
        if (!assetID) return;

        const route = state.routeLifecycleByVehicle.get(assetID);
        if (!route) return;

        const stops = Array.isArray(route.stops) ? route.stops : [];
        let selectedStopFound = false;
        for (const stop of stops) {
          const stopID = String(stop.id || "").trim();
          const lat = Number((stop.location || {}).latitude);
          const lon = Number((stop.location || {}).longitude);
          if (!stopID || !Number.isFinite(lat) || !Number.isFinite(lon)) continue;

          const status = String(stop.status || "pending");
          const marker = L.circleMarker([lat, lon], {
            radius: 5,
            color: stopStatusColor(status),
            fillColor: stopStatusColor(status),
            fillOpacity: 0.9,
            weight: 2
          }).addTo(map);
          marker.bindPopup(
            "<b>" + String(stop.name || stop.id || "stop") + "</b><br/>" +
            "Status: " + routeStatusText(status) + "<br/>" +
            "ETA: " + String(stop.etaTime || "") + "<br/>" +
            "Arrival: " + String(stop.arrivalTime || "-") + "<br/>" +
            "Departure: " + String(stop.departureTime || "-")
          );
          state.stopMarkers.set(stopID, {
            marker: marker,
            status: status
          });
          if (state.selectedStopID === stopID) {
            selectedStopFound = true;
          }
        }

        if (!selectedStopFound) {
          const currentStop = String(((route.progress || {}).currentStopId || "")).trim();
          const nextStop = String(((route.progress || {}).nextStopId || "")).trim();
          state.selectedStopID = currentStop || nextStop;
        }
        highlightSelectedStopMarker(false);
      }

      async function fetchJSON(path) {
        const headers = {};
        if (state.token.trim() !== "") {
          headers["Authorization"] = "Bearer " + state.token.trim();
        }
        headers["X-Samsara-Sim-Profile"] = "default";
        const res = await fetch(path, { headers: headers });
        if (!res.ok) {
          const text = await res.text();
          throw new Error("HTTP " + res.status + " " + text);
        }
        return res.json();
      }

      async function deleteJSON(path) {
        const headers = {};
        if (state.token.trim() !== "") {
          headers["Authorization"] = "Bearer " + state.token.trim();
        }
        headers["X-Samsara-Sim-Profile"] = "default";
        const res = await fetch(path, {
          method: "DELETE",
          headers: headers
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error("HTTP " + res.status + " " + text);
        }
        return res.json();
      }

      async function loadAssets() {
        const payload = await fetchJSON("/assets?limit=512");
        const list = Array.isArray(payload.data) ? payload.data : [];
        state.assetIDs = list.map((item) => String(item.id || "").trim()).filter(Boolean);
      }

      function chunk(values, size) {
        const out = [];
        for (let i = 0; i < values.length; i += size) {
          out.push(values.slice(i, i + size));
        }
        return out;
      }

      async function loadRouteGeometry() {
        const ids = state.assetIDs;
        if (ids.length === 0) {
          return [];
        }

        const all = [];
        const groups = chunk(ids, 40);
        for (const group of groups) {
          const params = new URLSearchParams();
          params.set("ids", group.join(","));
          const payload = await fetchJSON("/_sim/assets/routes?" + params.toString());
          const data = Array.isArray(payload.data) ? payload.data : [];
          for (const item of data) {
            all.push(item);
          }
        }
        return all;
      }

      async function loadRouteLifecycle() {
        const byVehicle = new Map();
        let after = "";
        for (let page = 0; page < 20; page++) {
          const params = new URLSearchParams();
          params.set("limit", "256");
          params.set("sortBy", "id");
          params.set("sortOrder", "asc");
          if (after !== "") {
            params.set("after", after);
          }
          const payload = await fetchJSON("/fleet/routes?" + params.toString());
          const data = Array.isArray(payload.data) ? payload.data : [];
          for (const route of data) {
            const vehicleID = String(((route || {}).vehicle || {}).id || "").trim();
            if (!vehicleID) continue;
            byVehicle.set(vehicleID, route);
          }

          const pagination = payload.pagination || {};
          const hasNext = Boolean(pagination.hasNextPage);
          after = String(pagination.endCursor || "").trim();
          if (!hasNext || after === "") {
            break;
          }
        }
        state.routeLifecycleByVehicle = byVehicle;
        updateRouteFilterChips();
        applyRouteVisibilityToPaths();
      }

      function renderRouteGeometry(routes) {
        for (const route of routes) {
          const assetID = String(route.assetId || "").trim();
          if (!assetID) continue;

          const points = Array.isArray(route.points) ? route.points : [];
          const coords = points
            .map((point) => [Number(point.latitude), Number(point.longitude)])
            .filter((pt) => Number.isFinite(pt[0]) && Number.isFinite(pt[1]));
          if (coords.length < 2) continue;

          const color = hashColor(assetID);
          if (!state.routePaths.has(assetID)) {
            const line = L.polyline(coords, {
              color: color,
              weight: 3,
              opacity: 0.55,
              dashArray: "5,7"
            }).addTo(map);
            state.routePaths.set(assetID, line);
          } else {
            state.routePaths.get(assetID).setLatLngs(coords);
          }
        }
        applyRouteVisibilityToPaths();
      }

      async function loadVehicleStats() {
        const ids = state.assetIDs;
        if (ids.length === 0) {
          return [];
        }

        const all = [];
        const groups = chunk(ids, 40);
        for (const group of groups) {
          const params = new URLSearchParams();
          params.set("vehicleIds", group.join(","));
          const payload = await fetchJSON("/fleet/vehicles/stats?" + params.toString());
          const data = Array.isArray(payload.data) ? payload.data : [];
          for (const item of data) {
            all.push(item);
          }
        }
        return all;
      }

      async function loadActiveEvents() {
        const params = new URLSearchParams();
        if (state.assetIDs.length > 0) {
          params.set("vehicleIds", state.assetIDs.join(","));
        }
        params.set("limit", "1024");
        const payload = await fetchJSON("/_sim/events/active?" + params.toString());
        const data = Array.isArray(payload.data) ? payload.data : [];

        const byVehicle = new Map();
        const byDriver = new Map();
        for (const event of data) {
          const vehicleID = String(event.vehicleId || "").trim();
          const driverID = String(event.driverId || "").trim();
          if (vehicleID) {
            if (!byVehicle.has(vehicleID)) byVehicle.set(vehicleID, []);
            byVehicle.get(vehicleID).push(event);
          }
          if (driverID) {
            if (!byDriver.has(driverID)) byDriver.set(driverID, []);
            byDriver.get(driverID).push(event);
          }
        }
        state.activeEventsByVehicle = byVehicle;
        state.activeEventsByDriver = byDriver;
      }

      async function loadWebhookInbox() {
        const payload = await fetchJSON("/_sim/webhooks/inbox?limit=40");
        const data = Array.isArray(payload.data) ? payload.data : [];
        state.webhookRecords = data;
      }

      function renderWebhookInbox() {
        const cards = [];
        for (const record of state.webhookRecords) {
          const eventType = String(record.eventType || "unknown").trim();
          const eventTime = String(record.eventTime || "").trim();
          const receivedAt = String(record.receivedAtTime || "").trim();
          const delivery = record.delivery || {};
          const signature = record.signature || {};
          const deliveryID = String(delivery.id || "").trim();
          const sequence = String(delivery.sequence || "").trim();
          const attempt = String(delivery.attempt || "").trim();
          const sigTime = String(signature.timestamp || "").trim();
          cards.push(
            "<div class='card'>" +
              "<div class='title'><span>" + eventType + "</span><span class='pill info'>" + (sequence === "" ? "seq -" : "seq " + sequence) + "</span></div>" +
              "<div class='kv'>" +
                "<div>Event Time</div><b>" + (eventTime || "-") + "</b>" +
                "<div>Received</div><b>" + (receivedAt || "-") + "</b>" +
                "<div>Attempt</div><b>" + (attempt || "-") + "</b>" +
                "<div>Delivery ID</div><b>" + (deliveryID || "-") + "</b>" +
                "<div>Sig Time</div><b>" + (sigTime || "-") + "</b>" +
              "</div>" +
            "</div>"
          );
        }
        if (cards.length === 0) {
          cards.push(
            "<div class='card'><div class='title'><span>No webhook deliveries yet</span></div><div class='kv'><div>Hint</div><b>Call /fleet/vehicles/stats or trigger events</b></div></div>"
          );
        }
        elWebhookList.innerHTML = cards.join("");
      }

      function renderAssets(stats) {
        const byID = new Map();
        for (const record of stats) {
          const assetID = String(record.id || "").trim();
          if (assetID) byID.set(assetID, record);
        }
        state.vehicleStats = byID;

        const bounds = [];
        const cards = [];
        const visibleAssets = new Set();
        for (const assetID of state.assetIDs) {
          const stat = byID.get(assetID);
          if (!stat) continue;

          const gps = stat.gps || {};
          const lat = Number(gps.latitude);
          const lon = Number(gps.longitude);
          if (!Number.isFinite(lat) || !Number.isFinite(lon)) continue;

          const route = state.routeLifecycleByVehicle.get(assetID) || {};
          const routeStatus = normalizeRouteStatus(route.status);
          const routeStatusTextValue = routeStatusText(routeStatus);
          const routeClass = routeStatusClass(routeStatus);
          const progress = Number(((route.progress || {}).percentComplete || 0));
          const progressText = Number.isFinite(progress) ? progress.toFixed(1) + "%" : "--";
          const nextStop = String(((route.progress || {}).nextStopId || "")).trim();
          const currentStop = String(((route.progress || {}).currentStopId || "")).trim();
          const isVisible = isRouteVisibleForFilter(routeStatus);

          const color = hashColor(assetID);

          if (!state.markers.has(assetID)) {
            const marker = L.circleMarker([lat, lon], {
              radius: 7,
              color: color,
              fillColor: color,
              fillOpacity: 0.95,
              weight: 2
            });
            state.markers.set(assetID, marker);
          }
          const marker = state.markers.get(assetID);
          marker.setLatLng([lat, lon]);

          if (isVisible) {
            if (!map.hasLayer(marker)) {
              marker.addTo(map);
            }
          } else {
            marker.remove();
          }

          marker.bindPopup(
            "<b>" + assetID + "</b><br/>" +
            "Lat: " + lat.toFixed(6) + "<br/>" +
            "Lon: " + lon.toFixed(6) + "<br/>" +
            "Speed mph: " + Number(gps.speedMilesPerHour || 0).toFixed(2) + "<br/>" +
              "Route: " + routeStatusTextValue + " (" + progressText + ")" + "<br/>" +
              "Time: " + String(gps.time || "")
          );
          rememberTrailPoint(assetID, lat, lon);
          if (isVisible && state.followSelected && state.selectedAssetID === assetID) {
            map.panTo([lat, lon], { animate: true, duration: 0.35 });
          }
          if (!isVisible) {
            continue;
          }
          visibleAssets.add(assetID);

          bounds.push([lat, lon]);
          const eventBadges = renderEventBadges(state.activeEventsByVehicle.get(assetID));
          const stopChips = renderStopChips(assetID, route);
          cards.push(
            "<div class='card interactive asset-card" +
              (state.selectedAssetID === assetID ? " selected" : "") +
              "' data-asset-id='" + assetID + "'>" +
              "<div class='title'><span>" + assetID + "</span><span class='pill " + routeClass + "'>" + routeStatusTextValue + "</span></div>" +
              eventBadges +
              "<div class='kv'>" +
                "<div>Speed</div><b>" + Number(gps.speedMilesPerHour || 0).toFixed(2) + " mph</b>" +
                "<div>Lat/Lon</div><b>" + lat.toFixed(5) + ", " + lon.toFixed(5) + "</b>" +
                "<div>Heading</div><b>" + Number(gps.headingDegrees || 0).toFixed(0) + "°</b>" +
                "<div>Progress</div><b>" + progressText + "</b>" +
                "<div>Current Stop</div><b>" + (currentStop || "-") + "</b>" +
                "<div>Next Stop</div><b>" + (nextStop || "-") + "</b>" +
                "<div>Time</div><b>" + String(gps.time || "") + "</b>" +
              "</div>" + stopChips +
            "</div>"
          );
        }

        if (state.selectedAssetID && !visibleAssets.has(state.selectedAssetID)) {
          state.selectedAssetID = "";
          state.selectedStopID = "";
          state.followSelected = false;
        }

        elAssetList.innerHTML = cards.join("");
        bindAssetCardClicks();
        bindStopChipClicks();
        applyRouteVisibilityToPaths();
        updateSelectedTrail();
        renderSelectedRouteStops();
        if (!state.fitDone && bounds.length > 1) {
          map.fitBounds(bounds, { padding: [28, 28] });
          state.fitDone = true;
        }
      }

      async function refreshHOS() {
        const payload = await fetchJSON("/fleet/hos/clocks?limit=512");
        const data = Array.isArray(payload.data) ? payload.data : [];
        const cards = [];
        for (const record of data) {
          const driver = record.driver || {};
          const driverID = String(driver.id || "").trim();
          const duty = record.currentDutyStatus || {};
          const vehicle = record.currentVehicle || {};
          const clocks = record.clocks || {};
          const drive = clocks.drive || {};
          const shift = clocks.shift || {};
          const cycle = clocks.cycle || {};
          const breakClock = clocks.break || {};
          const dutyType = String(duty.hosStatusType || "unknown");
          const className = dutyType === "driving" ? "ok" : "warn";
          const vehicleID = String(vehicle.id || "").trim();
          const stat = vehicleID ? state.vehicleStats.get(vehicleID) : null;
          const gps = stat && stat.gps ? stat.gps : {};
          const speedMph = Number(gps.speedMilesPerHour || 0);
          const activeEvents = state.activeEventsByDriver.get(driverID) || [];
          const isClickable = vehicleID !== "";
          const isSelected = vehicleID !== "" && state.selectedAssetID === vehicleID;
          cards.push(
            "<div class='card hos-card" +
              (isClickable ? " interactive" : "") +
              (isSelected ? " selected" : "") +
              "' data-driver-id='" + driverID + "' data-vehicle-id='" + vehicleID + "'>" +
              "<div class='title'><span>" + String(driver.name || driver.id || "Driver") + "</span><span class='pill " + className + "'>" + dutyType + "</span></div>" +
              renderEventBadges(activeEvents) +
              "<div class='kv'>" +
                "<div>Driver ID</div><b>" + String(driver.id || "") + "</b>" +
                "<div>Vehicle</div><b>" + (vehicleID || "unassigned") + "</b>" +
                "<div>Veh Speed</div><b>" + speedMph.toFixed(2) + " mph</b>" +
                "<div>Drive Rem</div><b>" + formatMS(drive.driveRemainingDurationMs) + "</b>" +
                "<div>Shift Rem</div><b>" + formatMS(shift.shiftRemainingDurationMs) + "</b>" +
                "<div>Break Rem</div><b>" + formatMS(breakClock.timeUntilBreakDurationMs) + "</b>" +
                "<div>Cycle Rem</div><b>" + formatMS(cycle.cycleRemainingDurationMs) + "</b>" +
              "</div>" +
            "</div>"
          );
        }
        elHOSList.innerHTML = cards.join("");
        bindHOSCardClicks();
      }

      async function tick() {
        if (!state.running || state.busy) return;
        state.busy = true;
        try {
          if (state.assetIDs.length === 0) {
            await loadAssets();
          }
          if (!state.routesLoaded) {
            const routes = await loadRouteGeometry();
            renderRouteGeometry(routes);
            state.routesLoaded = true;
          }

          const stats = await loadVehicleStats();
          await loadActiveEvents();
          await loadRouteLifecycle();
          renderAssets(stats);
          await refreshHOS();
          await loadWebhookInbox();
          renderWebhookInbox();
          let visibleAssets = 0;
          for (const marker of state.markers.values()) {
            if (map.hasLayer(marker)) visibleAssets++;
          }
          const tracking = state.selectedAssetID ? " | tracking " + state.selectedAssetID : "";
          const selectedRoute = state.selectedAssetID ? state.routeLifecycleByVehicle.get(state.selectedAssetID) : null;
          const routeSuffix = selectedRoute ? " | route " + routeStatusText(selectedRoute.status || "") : "";
          const filterSuffix = state.routeStatusFilter === "all" ?
            " | filter all" :
            " | filter " + routeStatusText(state.routeStatusFilter);
          const webhookSuffix = " | webhooks " + state.webhookRecords.length;
          setStatus(
            "updated " + new Date().toLocaleTimeString() +
            " | assets " + visibleAssets + "/" + state.assetIDs.length +
            filterSuffix + tracking + routeSuffix + webhookSuffix
          );
        } catch (err) {
          setStatus("error: " + err.message);
        } finally {
          state.busy = false;
        }
      }

      function restartTimer() {
        if (state.timer) {
          clearInterval(state.timer);
          state.timer = null;
        }
        if (state.running) {
          state.timer = setInterval(tick, state.pollMs);
        }
      }

      if (elRouteFilters) {
        elRouteFilters.addEventListener("click", function (event) {
          const chip = event.target.closest(".chip[data-route-filter]");
          if (!chip) return;
          const nextFilter = String(chip.getAttribute("data-route-filter") || "").trim();
          if (!nextFilter || nextFilter === state.routeStatusFilter) return;
          state.routeStatusFilter = nextFilter;
          updateRouteFilterChips();
          renderAssets(Array.from(state.vehicleStats.values()));
        });
      }

      btnApply.addEventListener("click", function () {
        state.token = String(elToken.value || "").trim();
        const poll = Number(elInterval.value || 1000);
        state.pollMs = Math.max(500, Number.isFinite(poll) ? poll : 1000);
        state.assetIDs = [];
        state.vehicleStats = new Map();
        state.routeLifecycleByVehicle = new Map();
        state.activeEventsByVehicle = new Map();
        state.activeEventsByDriver = new Map();
        state.webhookRecords = [];
        state.assetTrailPoints = new Map();
        state.routeStatusFilter = "all";
        state.selectedAssetID = "";
        state.selectedStopID = "";
        state.followSelected = false;
        state.selectedTrailAssetID = "";
        if (state.selectedTrailLine) {
          state.selectedTrailLine.remove();
          state.selectedTrailLine = null;
        }
        clearStopMarkers();
        state.routesLoaded = false;
        state.fitDone = false;
        for (const marker of state.markers.values()) marker.remove();
        for (const line of state.routePaths.values()) line.remove();
        state.markers.clear();
        state.routePaths.clear();
        elWebhookList.innerHTML = "";
        updateRouteFilterChips();
        setStatus("applying settings...");
        restartTimer();
        tick();
      });

      btnPause.addEventListener("click", function () {
        state.running = !state.running;
        btnPause.textContent = state.running ? "Pause" : "Resume";
        restartTimer();
      });

      btnClearWebhooks.addEventListener("click", async function () {
        if (state.busy) return;
        setStatus("clearing webhook inbox...");
        try {
          await deleteJSON("/_sim/webhooks/inbox");
          state.webhookRecords = [];
          renderWebhookInbox();
          setStatus("webhook inbox cleared");
        } catch (err) {
          setStatus("error: " + err.message);
        }
      });

      btnFocus.addEventListener("click", function () {
        if (state.selectedAssetID && state.markers.has(state.selectedAssetID)) {
          focusAsset(state.selectedAssetID, "focus");
          return;
        }
        const points = [];
        for (const entry of state.markers.values()) {
          if (!map.hasLayer(entry)) continue;
          const ll = entry.getLatLng();
          points.push([ll.lat, ll.lng]);
        }
        if (points.length > 1) {
          map.fitBounds(points, { padding: [28, 28] });
          return;
        }
        if (points.length === 1) {
          map.setView(points[0], 13);
        }
      });

      updateRouteFilterChips();
      restartTimer();
      tick();
    })();
  </script>
</body>
</html>`
