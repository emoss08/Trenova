import { AI_ROUTES } from "./aicontrol.mjs";
import { ORG, permissionManifest, USER } from "../fixtures/session.mjs";

const EMPTY_PAGE = { results: [], count: 0, next: null, prev: null };

const ROUTES = [
  ...AI_ROUTES,
  ["GET", /^\/api\/v1\/users\/me\/?$/, () => USER],
  ["GET", /^\/api\/v1\/me\/permissions\/?$/, () => permissionManifest()],
  ["GET", /^\/api\/v1\/me\/permissions\/version\/?$/, () => ({ checksum: "mock", expiresAt: Math.floor(Date.now() / 1000) + 86400 })],
  ["GET", /^\/api\/v1\/system\/version\/?$/, () => ({ version: "mock", commit: "mock", buildDate: "" })],
  ["GET", /^\/api\/v1\/system\/network-pulse\/?$/, () => ({ status: "ok" })],
  ["GET", /^\/api\/v1\/system\/public-config\/?$/, () => ({})],
  ["GET", /^\/api\/v1\/auth\/csrf\/?$/, () => ({ csrfToken: "mock-csrf", headerName: "X-CSRF-Token" })],
  [
    "GET",
    /^\/api\/v1\/users\/me\/organizations\/?$/,
    () => [{ id: ORG.id, name: ORG.name, city: "Dallas", state: "TX", logoUrl: null, isDefault: true, isCurrent: true }],
  ],
  [
    "GET",
    /^\/api\/v1\/integrations\/GoogleMaps\/runtime-config\/?$/,
    (state) => ({
      enabled: state.scenario.maps,
      configured: state.scenario.maps,
      ready: state.scenario.maps,
      missingRequiredFields: state.scenario.maps ? [] : ["apiKey"],
      config: state.scenario.maps ? { apiKey: "mock-key" } : {},
    }),
  ],
  [
    "GET",
    /^\/api\/v1\/integrations\/[A-Za-z]+\/runtime-config\/?$/,
    () => ({ enabled: false, configured: false, ready: false, missingRequiredFields: [], config: {} }),
  ],
  [
    "GET",
    /^\/api\/v1\/system\/update-status\/?$/,
    () => ({ currentVersion: "mock", updateAvailable: false, latestRelease: null, lastChecked: 0 }),
  ],
  ["GET", /^\/api\/v1\/page-favorites\/check\/?$/, () => ({ favorited: false })],
  ["GET", /^\/api\/v1\/assistant\/threads\/?$/, () => ({ items: [], total: 0 })],
  ["GET", /^\/api\/v1\/assistant\/turns\/active\/?$/, () => ({ items: [] })],
  ["GET", /^\/api\/v1\/documents\/uploads\/active\/?$/, () => []],
  ["GET", /^\/api\/v1\/realtime\/stream\/?$/, () => ({ __stream: true })],
  ["GET", /^\/api\/v1\/insights\/?$/, () => ({ items: [], total: 0 })],
  ["GET", /^\/api\/v1\/tables\/?$/, () => ({ results: [], count: 0 })],
];

export function handleRest(state, { method, path }) {
  for (const [routeMethod, pattern, handler] of ROUTES) {
    if (routeMethod === method && pattern.test(path)) {
      const payload = handler(state, path);
      if (payload && payload.__stream) return { known: true, stream: true };
      return { known: true, payload };
    }
  }
  if (method === "GET") {
    return { known: false, status: 200, payload: EMPTY_PAGE };
  }
  return { known: false, status: 200, payload: {} };
}

export { ORG };
