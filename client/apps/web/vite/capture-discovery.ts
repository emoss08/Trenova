import type { Plugin } from "vite";

// Trenova Capture is given the address people open Trenova at in a browser.
// On a self-hosted install the API answers under that same address; on
// Trenova Cloud, and on any deployment that serves the web app and the API
// from different origins, it does not. The web app therefore publishes where
// its API is, at a well-known path the companion reads before it pairs:
//
//   GET /.well-known/trenova-capture.json
//   {"product":"trenova","apiBaseUrl":"https://api.example.com"}
//
// The address comes from the same VITE_API_URL the app itself calls, with the
// versioned prefix taken off, so the two can never disagree. A relative API
// base ("/api/v1", the self-hosted edge) publishes "/", which the companion
// resolves against the address it was given.

export const CAPTURE_WELL_KNOWN_PATH = ".well-known/trenova-capture.json";

const SERVER_API_PREFIX = "/api/v1";

export type CaptureWellKnown = {
  product: "trenova";
  apiBaseUrl: string;
};

/**
 * The server address a companion should use, from the API base the app was
 * built with: the origin and any mount path before `/api/v1`.
 */
export function captureApiBaseUrl(apiBase: string): string {
  const trimmed = apiBase.trim().replace(/\/+$/, "");
  const mount = trimmed.endsWith(SERVER_API_PREFIX)
    ? trimmed.slice(0, -SERVER_API_PREFIX.length)
    : trimmed;
  if (/^https?:\/\//i.test(mount)) {
    return mount;
  }
  return mount === "" ? "/" : mount.startsWith("/") ? mount : `/${mount}`;
}

export function captureWellKnown(apiBase: string): CaptureWellKnown {
  return { product: "trenova", apiBaseUrl: captureApiBaseUrl(apiBase) };
}

/** The file as written: one line, stable, so a proxy can cache it. */
export function captureWellKnownBody(apiBase: string): string {
  return `${JSON.stringify(captureWellKnown(apiBase))}\n`;
}

/**
 * Emits the well-known file into the build and serves it from the dev server.
 * The API base is the resolved VITE_API_URL, or `/api/v1` when the build sets
 * none, matching `resolveApiBaseUrl` in @trenova/shared/lib/constants.
 */
export function captureDiscovery(): Plugin {
  let apiBase = SERVER_API_PREFIX;

  return {
    name: "trenova:capture-discovery",
    configResolved(config) {
      const configured = config.env.VITE_API_URL;
      if (typeof configured === "string" && configured.trim() !== "") {
        apiBase = configured;
      } else if (config.mode === "development") {
        apiBase = "http://localhost:8080/api/v1";
      }
    },
    configureServer(server) {
      server.middlewares.use(`/${CAPTURE_WELL_KNOWN_PATH}`, (_request, response) => {
        response.setHeader("Content-Type", "application/json; charset=utf-8");
        response.setHeader("Cache-Control", "no-cache");
        response.end(captureWellKnownBody(apiBase));
      });
    },
    generateBundle() {
      this.emitFile({
        type: "asset",
        fileName: CAPTURE_WELL_KNOWN_PATH,
        source: captureWellKnownBody(apiBase),
      });
    },
  };
}
