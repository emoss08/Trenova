const scriptSources = [
  "'self'",
  "'unsafe-eval'",
  "https://static.cloudflareinsights.com",
  "'sha256-Q9qAP4vtJuwS7pBhw9g2oS9FueKw67t+u398X99GROo='",
  "'sha256-XtR73bEqMUD7aevUCpctukznhxuFL3vHjrYpUg9FkbI='",
  "https://maps.googleapis.com",
  "https://maps.gstatic.com",
] as const;
const localDevelopmentScriptSources = [
  "'self'",
  "'unsafe-eval'",
  "'unsafe-inline'",
  "https://static.cloudflareinsights.com",
  "https://maps.googleapis.com",
  "https://maps.gstatic.com",
] as const;
const styleSources = ["'self'", "'unsafe-inline'", "https://fonts.googleapis.com"] as const;
const fontSources = ["'self'", "data:", "https://fonts.gstatic.com"] as const;
const imageSources = [
  "'self'",
  "data:",
  "blob:",
  "https://maps.googleapis.com",
  "https://maps.gstatic.com",
  "https://*.googleapis.com",
  "https://*.gstatic.com",
  "https://*.googleusercontent.com",
  "https://tilecache.rainviewer.com",
  "https://tile.openweathermap.org",
  "https://storage.trenova.app",
] as const;
const localDevelopmentImageSources = [
  "http://localhost:9000",
  "http://127.0.0.1:9000",
] as const;
const connectSources = [
  "'self'",
  "https://api.trenova.app",
  "https://api.rainviewer.com",
  "https://cloudflareinsights.com",
  "https://storage.trenova.app",
  "https://maps.googleapis.com",
  "https://*.googleapis.com",
  "https://*.gstatic.com",
  "https://*.ably.io",
  "https://*.ably.net",
  "https://*.ably-realtime.com",
  "wss://*.ably.io",
  "wss://*.ably.net",
  "wss://*.ably-realtime.com",
] as const;
const localDevelopmentConnectSources = [
  "http://localhost:*",
  "http://127.0.0.1:*",
  "ws://localhost:*",
  "ws://127.0.0.1:*",
] as const;

const baseSecurityHeaders = {
  "X-Content-Type-Options": "nosniff",
  "X-Frame-Options": "DENY",
  "Referrer-Policy": "strict-origin-when-cross-origin",
  "Permissions-Policy": "camera=(), microphone=(), geolocation=()",
} as const;

const sensitivePathPrefixes = [
  "/api/",
  "/auth/",
  "/debug/",
  "/swagger/",
  "/.git",
  "/.env",
  "/config",
  "/openapi",
] as const;

const sensitiveExactPaths = ["/api", "/auth", "/metrics", "/debug", "/swagger"] as const;

type StaticAssetBinding = {
  fetch(request: Request): Promise<Response>;
};

type WorkerEnv = {
  ASSETS: StaticAssetBinding;
};

type WorkerModule = {
  fetch(request: Request, env: WorkerEnv): Promise<Response>;
};

export const worker: WorkerModule = {
  async fetch(request: Request, env: WorkerEnv): Promise<Response> {
    const url = new URL(request.url);

    if (isLocalDevelopmentRequest(request) && isLocalDevelopmentAPIPath(url.pathname)) {
      return fetchLocalDevelopmentAPI(request);
    }

    if (!isStaticAssetMethod(request.method)) {
      return notFoundResponse(request);
    }

    if (isBlockedPath(url.pathname)) {
      return notFoundResponse(request);
    }

    if (isFileLikePath(url.pathname)) {
      const response = await env.ASSETS.fetch(assetLookupRequest(request));

      if (response.status === 404) {
        return notFoundResponse(request);
      }

      return withSecurityHeaders(request, response);
    }

    if (!isBrowserNavigation(request)) {
      const response = await env.ASSETS.fetch(request);

      if (response.status === 404) {
        return notFoundResponse(request);
      }

      return withSecurityHeaders(request, response);
    }

    return withSecurityHeaders(request, await env.ASSETS.fetch(request));
  },
};

export default worker;

function isStaticAssetMethod(method: string): boolean {
  const normalizedMethod = method.toUpperCase();
  return normalizedMethod === "GET" || normalizedMethod === "HEAD";
}

function isBlockedPath(pathname: string): boolean {
  const normalizedPathname = pathname.toLowerCase();

  if (normalizedPathname.endsWith(".map")) {
    return true;
  }

  if (sensitiveExactPaths.includes(normalizedPathname as (typeof sensitiveExactPaths)[number])) {
    return true;
  }

  return sensitivePathPrefixes.some((prefix) => normalizedPathname.startsWith(prefix));
}

function isLocalDevelopmentAPIPath(pathname: string): boolean {
  const normalizedPathname = pathname.toLowerCase();
  return normalizedPathname === "/api" || normalizedPathname.startsWith("/api/");
}

async function fetchLocalDevelopmentAPI(request: Request): Promise<Response> {
  const targetURL = localDevelopmentAPIURL(request);
  if (!targetURL) {
    return notFoundResponse(request);
  }

  const headers = new Headers(request.headers);
  headers.delete("Host");

  const hasBody = request.method !== "GET" && request.method !== "HEAD";

  return fetch(
    new Request(targetURL, {
      body: hasBody ? request.body : null,
      headers,
      method: request.method,
      redirect: "manual",
    }),
  );
}

function localDevelopmentAPIURL(request: Request): URL | null {
  const configuredURL = (import.meta.env.VITE_API_URL as string | undefined) ?? "";
  if (!configuredURL) {
    return null;
  }

  try {
    const requestURL = new URL(request.url);
    const apiURL = new URL(configuredURL);
    apiURL.pathname = requestURL.pathname;
    apiURL.search = requestURL.search;
    apiURL.hash = "";

    return apiURL;
  } catch {
    return null;
  }
}

function isFileLikePath(pathname: string): boolean {
  const lastSegment = pathname.split("/").pop() ?? "";
  return /\.[a-z0-9][a-z0-9-]*$/i.test(lastSegment);
}

function isBrowserNavigation(request: Request): boolean {
  const fetchMode = request.headers.get("Sec-Fetch-Mode")?.toLowerCase();
  if (fetchMode === "navigate") {
    return true;
  }

  const acceptHeader = request.headers.get("Accept")?.toLowerCase() ?? "";
  return acceptHeader.includes("text/html");
}

function assetLookupRequest(request: Request): Request {
  const headers = new Headers(request.headers);
  headers.set("Accept", "*/*");
  headers.delete("Sec-Fetch-Mode");

  return new Request(request, { headers });
}

function notFoundResponse(request: Request): Response {
  return withSecurityHeaders(request, new Response(null, { status: 404 }));
}

function withSecurityHeaders(request: Request | null, response: Response): Response {
  const headers = new Headers(response.headers);

  for (const [header, value] of Object.entries(baseSecurityHeaders)) {
    headers.set(header, value);
  }
  if (!request || !isLocalDevelopmentRequest(request)) {
    headers.set("Strict-Transport-Security", "max-age=31536000; includeSubDomains");
  }
  headers.set("Content-Security-Policy", contentSecurityPolicy(request));

  return new Response(response.body, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}

function contentSecurityPolicy(request: Request | null): string {
  const isLocalDevelopment = request ? isLocalDevelopmentRequest(request) : false;
  const effectiveScriptSources = isLocalDevelopment
    ? [...localDevelopmentScriptSources]
    : [...scriptSources];
  const effectiveConnectSources = isLocalDevelopment
    ? [...connectSources, ...localDevelopmentConnectSources]
    : [...connectSources];
  const effectiveImageSources = isLocalDevelopment
    ? [...imageSources, ...localDevelopmentImageSources]
    : [...imageSources];

  return [
    "default-src 'self'",
    "base-uri 'self'",
    "object-src 'none'",
    "frame-ancestors 'none'",
    "form-action 'self'",
    `script-src ${effectiveScriptSources.join(" ")}`,
    `style-src ${styleSources.join(" ")}`,
    `font-src ${fontSources.join(" ")}`,
    `img-src ${effectiveImageSources.join(" ")}`,
    `connect-src ${effectiveConnectSources.join(" ")}`,
    "worker-src 'self' blob:",
    "manifest-src 'self'",
    "upgrade-insecure-requests",
  ].join("; ");
}

function isLocalDevelopmentRequest(request: Request): boolean {
  const hostname = new URL(request.url).hostname.toLowerCase();
  return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]";
}
