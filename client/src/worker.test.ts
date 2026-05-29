import { afterEach, describe, expect, it, vi } from "vitest";
import worker from "./worker";

type AssetFixture = {
  body: BodyInit | null;
  headers?: HeadersInit;
  status?: number;
};

const defaultIndexFixture: AssetFixture = {
  body: "<!doctype html><html><body>Trenova</body></html>",
  headers: { "Content-Type": "text/html; charset=utf-8" },
};

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("Cloudflare SPA worker", () => {
  it("serves React navigation paths with security headers", async () => {
    const env = createAssetEnv();
    const request = new Request("https://cloud.trenova.app/shipments", {
      headers: { Accept: "text/html" },
    });

    const response = await worker.fetch(request, env);

    expect(response.status).toBe(200);
    expect(await response.text()).toContain("Trenova");
    expectSecurityHeaders(response.headers);
  });

  it("relaxes only localhost development CSP requirements for Vite", async () => {
    const env = createAssetEnv();
    const request = new Request("http://127.0.0.1:5174/shipments", {
      headers: { Accept: "text/html" },
    });

    const response = await worker.fetch(request, env);

    expect(response.status).toBe(200);
    expect(response.headers.get("Strict-Transport-Security")).toBeNull();
    expect(response.headers.get("Content-Security-Policy")).toContain("'unsafe-inline'");
    expect(response.headers.get("Content-Security-Policy")).not.toContain("sha256-");
    expect(response.headers.get("Content-Security-Policy")).toContain("ws://127.0.0.1:*");
    expect(response.headers.get("Content-Security-Policy")).toContain("http://localhost:*");
    expect(response.headers.get("Content-Security-Policy")).toContain("http://localhost:8080");
    expect(response.headers.get("Content-Security-Policy")).toContain("http://127.0.0.1:8080");
    expect(response.headers.get("Content-Security-Policy")).toContain("http://localhost:9000");
    expect(response.headers.get("Content-Security-Policy")).toContain("http://127.0.0.1:9000");
  });

  it("returns 404 for sensitive paths before static asset lookup", async () => {
    const env = createAssetEnv();
    const blockedPaths = [
      "/metrics",
      "/debug/pprof/",
      "/openapi.json",
      "/config.json",
      "/.git/config",
      "/.env.production",
      "/auth/login",
      "/assets/index.js.map",
    ];

    for (const path of blockedPaths) {
      const response = await worker.fetch(
        new Request(`https://cloud.trenova.app${path}`, {
          headers: { Accept: "text/html" },
        }),
        env,
      );

      expect(response.status, path).toBe(404);
      expectSecurityHeaders(response.headers);
    }

    expect(env.ASSETS.fetch).not.toHaveBeenCalled();
  });

  it("blocks API paths on production hosts", async () => {
    const env = createAssetEnv();
    const response = await worker.fetch(
      new Request("https://cloud.trenova.app/api/v1/auth/login", {
        method: "POST",
        headers: { Accept: "application/json" },
      }),
      env,
    );

    expect(response.status).toBe(404);
    expect(env.ASSETS.fetch).not.toHaveBeenCalled();
  });

  it("passes API paths through to the local backend in development", async () => {
    const env = createAssetEnv();
    vi.stubEnv("VITE_API_URL", "http://localhost:8080/api/v1");

    const fetchMock = vi.fn(async (_request: Request) => {
      return new Response(JSON.stringify({ ok: true }), {
        headers: { "Content-Type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const response = await worker.fetch(
      new Request("http://127.0.0.1:5174/api/v1/auth/login?next=%2Fshipments", {
        body: JSON.stringify({ emailAddress: "test@example.com", password: "password" }),
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": "dev-csrf-token",
        },
        method: "POST",
      }),
      env,
    );

    expect(response.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledTimes(1);

    const forwardedRequest = fetchMock.mock.calls[0]?.[0];
    expect(forwardedRequest).toBeDefined();
    if (!forwardedRequest) {
      throw new Error("expected forwarded API request");
    }

    expect(forwardedRequest.url).toBe("http://localhost:8080/api/v1/auth/login?next=%2Fshipments");
    expect(forwardedRequest.method).toBe("POST");
    expect(forwardedRequest.headers.get("X-CSRF-Token")).toBe("dev-csrf-token");
    expect(await forwardedRequest.text()).toBe(
      JSON.stringify({ emailAddress: "test@example.com", password: "password" }),
    );
  });

  it("returns 404 for local API paths when VITE_API_URL is not absolute", async () => {
    const env = createAssetEnv();
    vi.stubEnv("VITE_API_URL", "/api/v1");

    const response = await worker.fetch(
      new Request("http://127.0.0.1:5174/api/v1/auth/login", {
        method: "POST",
        headers: { Accept: "application/json" },
      }),
      env,
    );

    expect(response.status).toBe(404);
    expect(env.ASSETS.fetch).not.toHaveBeenCalled();
  });

  it("returns 404 for missing file-like paths instead of SPA HTML", async () => {
    const env = createAssetEnv({
      fallbackForNavigations: true,
      fixtures: {
        "/logo.ico": {
          body: "icon",
          headers: { "Content-Type": "image/x-icon" },
        },
      },
    });

    const response = await worker.fetch(
      new Request("https://cloud.trenova.app/manifest.json", {
        headers: {
          Accept: "text/html",
          "Sec-Fetch-Mode": "navigate",
        },
      }),
      env,
    );

    expect(response.status).toBe(404);
    expectSecurityHeaders(response.headers);

    const assetRequest = env.ASSETS.fetch.mock.calls[0]?.[0] as Request | undefined;
    expect(assetRequest?.headers.get("Accept")).toBe("*/*");
    expect(assetRequest?.headers.has("Sec-Fetch-Mode")).toBe(false);
  });

  it("serves real file-like assets with their content types and security headers", async () => {
    const env = createAssetEnv({
      fixtures: {
        "/logo.ico": {
          body: "icon",
          headers: { "Content-Type": "image/x-icon" },
        },
        "/assets/js/index.abc123.js": {
          body: "console.log('loaded');",
          headers: { "Content-Type": "application/javascript; charset=utf-8" },
        },
        "/assets/index.def456.css": {
          body: "body{margin:0}",
          headers: { "Content-Type": "text/css; charset=utf-8" },
        },
      },
    });

    const tests = [
      { path: "/logo.ico", contentType: "image/x-icon" },
      { path: "/assets/js/index.abc123.js", contentType: "application/javascript; charset=utf-8" },
      { path: "/assets/index.def456.css", contentType: "text/css; charset=utf-8" },
    ];

    for (const tt of tests) {
      const response = await worker.fetch(
        new Request(`https://cloud.trenova.app${tt.path}`, {
          headers: { Accept: "*/*" },
        }),
        env,
      );

      expect(response.status, tt.path).toBe(200);
      expect(response.headers.get("Content-Type")).toBe(tt.contentType);
      expectSecurityHeaders(response.headers);
    }
  });

  it("rejects unsupported methods", async () => {
    const env = createAssetEnv();
    const response = await worker.fetch(
      new Request("https://cloud.trenova.app/shipments", {
        method: "POST",
        headers: { Accept: "text/html" },
      }),
      env,
    );

    expect(response.status).toBe(404);
    expectSecurityHeaders(response.headers);
    expect(env.ASSETS.fetch).not.toHaveBeenCalled();
  });
});

function createAssetEnv(
  options: {
    fallbackForNavigations?: boolean;
    fixtures?: Record<string, AssetFixture>;
  } = {},
) {
  const fixtures = options.fixtures ?? {};

  return {
    ASSETS: {
      fetch: vi.fn(async (request: Request) => {
        const url = new URL(request.url);
        const fixture = fixtures[url.pathname];
        if (fixture) {
          return new Response(fixture.body, {
            status: fixture.status ?? 200,
            headers: fixture.headers,
          });
        }

        if (
          options.fallbackForNavigations &&
          request.headers.get("Sec-Fetch-Mode") === "navigate"
        ) {
          return new Response(defaultIndexFixture.body, {
            status: 200,
            headers: defaultIndexFixture.headers,
          });
        }

        if (url.pathname === "/" || isNavigationRequest(request)) {
          return new Response(defaultIndexFixture.body, {
            status: 200,
            headers: defaultIndexFixture.headers,
          });
        }

        return new Response(null, { status: 404 });
      }),
    },
  };
}

function isNavigationRequest(request: Request): boolean {
  return request.headers.get("Accept")?.includes("text/html") ?? false;
}

function expectSecurityHeaders(headers: Headers): void {
  expect(headers.get("Strict-Transport-Security")).toBe("max-age=31536000; includeSubDomains");
  expect(headers.get("X-Content-Type-Options")).toBe("nosniff");
  expect(headers.get("X-Frame-Options")).toBe("DENY");
  expect(headers.get("Referrer-Policy")).toBe("strict-origin-when-cross-origin");
  expect(headers.get("Permissions-Policy")).toBe("camera=(), microphone=(), geolocation=()");
  expect(headers.get("Content-Security-Policy")).toContain("default-src 'self'");
  expect(headers.get("Content-Security-Policy")).toContain("https://api.trenova.app");
  expect(headers.get("Content-Security-Policy")).toContain("https://static.cloudflareinsights.com");
  expect(headers.get("Content-Security-Policy")).toContain(
    "'sha256-Q9qAP4vtJuwS7pBhw9g2oS9FueKw67t+u398X99GROo='",
  );
  expect(headers.get("Content-Security-Policy")).toContain(
    "'sha256-XtR73bEqMUD7aevUCpctukznhxuFL3vHjrYpUg9FkbI='",
  );
  expect(headers.get("Content-Security-Policy")).toContain("https://cloudflareinsights.com");
  expect(headers.get("Content-Security-Policy")).toContain("https://storage.trenova.app");
  expect(headers.get("Content-Security-Policy")).not.toContain("http://localhost:8080");
  expect(headers.get("Content-Security-Policy")).not.toContain("http://127.0.0.1:8080");
  expect(headers.get("Content-Security-Policy")).not.toContain("http://localhost:9000");
  expect(headers.get("Content-Security-Policy")).not.toContain("http://127.0.0.1:9000");
  expect(headers.get("Content-Security-Policy")).toContain("https://tilecache.rainviewer.com");
  expect(headers.get("Content-Security-Policy")).toContain("https://tile.openweathermap.org");
  expect(headers.get("Content-Security-Policy")).toContain("https://*.ably.net");
  expect(headers.get("Content-Security-Policy")).toContain("wss://*.ably.net");
  expect(headers.get("Content-Security-Policy")).toContain("wss://*.ably-realtime.com");
}
