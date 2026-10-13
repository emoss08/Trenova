// The web app's service worker, bundled into /sw.js by vite/service-worker.ts with the build's
// manifest in place of __SHELL__. It keeps two caches:
//
//   trenova-shell-<version>  the HTML shell of one build, so a returning visit opens the app
//                            without waiting on the network; replaced by each new build.
//   trenova-assets           content-hashed build files, kept across builds and trimmed to the
//                            current build's files when a new worker takes over.
//
// A new build ships a new sw.js; the browser installs it in the background and it takes over
// at once. The page then moves onto the new build at its next change of page
// (lib/service-worker), and a page still running the old build that asks for a file the new
// build dropped reloads into it (lib/chunk-recovery).
//
// The caches are a speed-up and never a dependency: whenever the browser's cache storage fails
// (quota, eviction, a private mode), the request goes to the network as if there were no
// worker.

import { shellLoadsBuild, type ShellManifest } from "./manifest";
import { isCacheableAsset, routeRequest } from "./routing";

declare const self: ServiceWorkerGlobalScope;

// null when the worker is switched off (VITE_SERVICE_WORKER=off): it then removes itself.
declare const __SHELL__: ShellManifest | null;

const SHELL_PREFIX = "trenova-shell-";
const ASSET_CACHE = "trenova-assets";

if (__SHELL__ === null) {
  retire();
} else {
  serve(__SHELL__);
}

function serve(manifest: ShellManifest): void {
  const shellCache = `${SHELL_PREFIX}${manifest.version}`;

  self.addEventListener("install", (event) => {
    event.waitUntil(
      (async () => {
        await cacheShell(manifest, shellCache);
        await cacheStartupFiles(manifest);
        await self.skipWaiting();
      })(),
    );
  });

  self.addEventListener("activate", (event) => {
    event.waitUntil(
      (async () => {
        await dropOtherBuilds(manifest, shellCache);
        await self.clients.claim();
      })(),
    );
  });

  self.addEventListener("fetch", (event) => {
    const route = routeRequest(event.request, self.location.origin, manifest.base);
    if (route === "asset") event.respondWith(cachedAsset(event));
    else if (route === "shell") event.respondWith(cachedShell(event.request, shellCache, manifest));
  });
}

// An install that throws leaves the previous worker, and the shell it serves, in place; the
// browser installs this one again at its next check.
async function cacheShell(manifest: ShellManifest, shellCache: string): Promise<void> {
  const response = await fetch(new Request(manifest.base, { cache: "reload" }));
  if (response.status !== 200 || response.redirected) {
    throw new Error(`The app shell answered ${response.status}; not caching it`);
  }
  const html = await response.clone().text();
  if (!shellLoadsBuild(html, manifest)) {
    throw new Error("The app shell is from another build; not caching it");
  }
  await (await caches.open(shellCache)).put(manifest.base, response);
}

async function cacheStartupFiles(manifest: ShellManifest): Promise<void> {
  const assets = await caches.open(ASSET_CACHE);
  for (const url of manifest.startup) {
    if (await assets.match(url)) continue;
    const response = await fetch(url);
    if (!isCacheableAsset(response)) {
      throw new Error(`The build file ${url} is not on the server yet; not installing`);
    }
    await assets.put(url, response);
  }
}

async function dropOtherBuilds(manifest: ShellManifest, shellCache: string): Promise<void> {
  for (const name of await caches.keys()) {
    if (name.startsWith(SHELL_PREFIX) && name !== shellCache) await caches.delete(name);
  }
  const current = new Set(manifest.assets);
  const assets = await caches.open(ASSET_CACHE);
  for (const request of await assets.keys()) {
    if (!current.has(new URL(request.url).pathname)) await assets.delete(request);
  }
}

async function cachedAsset(event: FetchEvent): Promise<Response> {
  const hit = await fromCache(ASSET_CACHE, event.request);
  if (hit) return hit;

  const response = await fetch(event.request);
  if (isCacheableAsset(response)) {
    const copy = response.clone();
    event.waitUntil(
      caches
        .open(ASSET_CACHE)
        .then((cache) => cache.put(event.request, copy))
        .catch(() => undefined),
    );
  }
  return response;
}

async function cachedShell(
  request: Request,
  shellCache: string,
  manifest: ShellManifest,
): Promise<Response> {
  return (await fromCache(shellCache, manifest.base)) ?? fetch(request);
}

async function fromCache(cacheName: string, request: RequestInfo): Promise<Response | undefined> {
  try {
    return await (await caches.open(cacheName)).match(request);
  } catch {
    return undefined;
  }
}

// Switched off: answer nothing, delete what the worker kept, unregister, and reload the pages
// it controlled so they run without it.
function retire(): void {
  self.addEventListener("install", (event) => event.waitUntil(self.skipWaiting()));
  self.addEventListener("activate", (event) => {
    event.waitUntil(
      (async () => {
        for (const name of await caches.keys()) {
          if (name.startsWith(SHELL_PREFIX) || name === ASSET_CACHE) await caches.delete(name);
        }
        await self.registration.unregister();
        for (const client of await self.clients.matchAll({ type: "window" })) {
          await client.navigate(client.url);
        }
      })(),
    );
  });
}
