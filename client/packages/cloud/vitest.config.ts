import path from "node:path";
import { defineConfig } from "vitest/config";
import { editionAlias } from "../edition/node/resolve-entry.ts";

const dirname = import.meta.dirname;
const webRoot = path.resolve(dirname, "../../apps/web");

// The overlay is compiled into the web app, so its tests resolve exactly as the web
// app's do: `@/` is the host's src, shared is aliased to its source, and the edition
// entry resolves to this package.
export default defineConfig({
  resolve: {
    alias: {
      "@": path.join(webRoot, "src"),
      "@trenova/shared": path.resolve(dirname, "../shared/src"),
      ...editionAlias(),
    },
    dedupe: ["react-router"],
  },
  test: {
    name: "cloud",
    environment: "happy-dom",
    setupFiles: [path.join(webRoot, "src/test-setup.ts")],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
