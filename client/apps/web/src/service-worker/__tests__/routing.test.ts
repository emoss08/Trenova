import { describe, expect, it } from "vitest";
import { isCacheableAsset, routeRequest, type RoutedRequest } from "../routing";

const ORIGIN = "https://app.trenova.example";

function request(
  path: string,
  {
    method = "GET",
    mode = "cors",
    accept = "*/*",
  }: { method?: string; mode?: string; accept?: string } = {},
): RoutedRequest {
  return {
    method,
    mode,
    url: path.startsWith("http") ? path : `${ORIGIN}${path}`,
    headers: { get: (name) => (name.toLowerCase() === "accept" ? accept : null) },
  };
}

const route = (r: RoutedRequest) => routeRequest(r, ORIGIN, "/");

describe("routeRequest", () => {
  it("serves the app's content-hashed build files from the cache", () => {
    expect(route(request("/assets/js/main.DcyZLU6s.js"))).toBe("asset");
    expect(route(request("/assets/initial-styles.B3lfDv2s.css"))).toBe("asset");
  });

  it("answers a page of the app with the HTML shell", () => {
    expect(route(request("/", { mode: "navigate" }))).toBe("shell");
    expect(route(request("/shipments/shp_1?tab=stops", { mode: "navigate" }))).toBe("shell");
    expect(route(request("/carrier-tracking/abc123", { mode: "navigate" }))).toBe("shell");
  });

  it("never touches the API, GraphQL or their event streams", () => {
    expect(route(request("/api/v1/shipments/"))).toBe("network");
    expect(route(request("/api/v1/documents/doc_1/download/", { mode: "navigate" }))).toBe(
      "network",
    );
    expect(route(request("/graphql", { method: "POST" }))).toBe("network");
    expect(route(request("/graphql"))).toBe("network");
    expect(route(request("/api/v1/realtime/stream/", { accept: "text/event-stream" }))).toBe(
      "network",
    );
  });

  it("leaves Dash, the well-known files and other origins alone", () => {
    expect(route(request("/dash/", { mode: "navigate" }))).toBe("network");
    expect(route(request("/dash/assets/index.abc.js"))).toBe("network");
    expect(route(request("/.well-known/trenova-capture.json"))).toBe("network");
    expect(route(request("https://maps.googleapis.com/maps/api/js"))).toBe("network");
    expect(route(request("https://api.trenova.example/api/v1/me/"))).toBe("network");
  });

  it("leaves files without a content hash to HTTP caching", () => {
    expect(route(request("/assets/fonts/inter-latin-wght-normal.woff2"))).toBe("network");
    expect(route(request("/logo.ico", { mode: "navigate" }))).toBe("network");
    expect(route(request("/sw.js"))).toBe("network");
    expect(route(request("/cmaps/UniKS-UTF32-H.bcmap"))).toBe("network");
  });

  it("only ever answers reads", () => {
    expect(route(request("/assets/js/main.DcyZLU6s.js", { method: "POST" }))).toBe("network");
    expect(route(request("/", { method: "HEAD", mode: "navigate" }))).toBe("network");
  });

  it("keeps to the app's mount path", () => {
    const mounted = (r: RoutedRequest) => routeRequest(r, ORIGIN, "/app/");
    expect(mounted(request("/app/assets/js/main.a.js"))).toBe("asset");
    expect(mounted(request("/app/shipments", { mode: "navigate" }))).toBe("shell");
    expect(mounted(request("/other", { mode: "navigate" }))).toBe("network");
    expect(mounted(request("/app/api/v1/me/"))).toBe("network");
  });
});

describe("isCacheableAsset", () => {
  const response = (status: number, contentType: string, type = "basic") => ({
    status,
    type,
    headers: {
      get: (name: string) => (name.toLowerCase() === "content-type" ? contentType : null),
    },
  });

  it("keeps a build file the server returned", () => {
    expect(isCacheableAsset(response(200, "text/javascript"))).toBe(true);
    expect(isCacheableAsset(response(200, "text/css; charset=utf-8"))).toBe(true);
  });

  it("refuses the HTML a single-page-app host returns, with a 200, for a missing file", () => {
    expect(isCacheableAsset(response(200, "text/html"))).toBe(false);
    expect(isCacheableAsset(response(200, "text/html; charset=utf-8"))).toBe(false);
  });

  it("refuses errors, partial and cross-origin responses", () => {
    expect(isCacheableAsset(response(404, "text/plain"))).toBe(false);
    expect(isCacheableAsset(response(206, "text/javascript"))).toBe(false);
    expect(isCacheableAsset(response(200, "text/javascript", "opaque"))).toBe(false);
  });
});
