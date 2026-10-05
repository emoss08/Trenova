import { copyFile } from "node:fs/promises";
import path from "node:path";
import type { Plugin } from "vite";

// Removed from client/apps/dash/vite.config.ts with the Cloudflare deploy. The app is
// served under /dash, so the bundle is emitted to dist/dash and a copy of index.html is
// placed at the dist root so Cloudflare's single-page-application not-found handling
// can resolve deep links. Add it back to the Dash build's plugins when deploying with
// wrangler.jsonc beside this file; `dashRoot` is client/apps/dash.
export function rootIndexFallback(dashRoot: string): Plugin {
  return {
    name: "dash-root-index-fallback",
    apply: "build",
    async closeBundle() {
      await copyFile(
        path.resolve(dashRoot, "dist/dash/index.html"),
        path.resolve(dashRoot, "dist/index.html"),
      );
    },
  };
}
