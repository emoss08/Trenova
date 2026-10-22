import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  // rules/test-module.ts imports rules.wasm?inline so the setup can load it without node:fs.
  assetsInclude: ["**/*.wasm"],
  resolve: {
    alias: {
      "@trenova/shared": path.resolve(dirname, "./src"),
    },
  },
  test: {
    name: "shared",
    environment: "happy-dom",
    setupFiles: ["./src/test-setup.ts"],
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
  },
});
