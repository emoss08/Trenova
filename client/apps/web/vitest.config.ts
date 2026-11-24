import tailwindcss from "@tailwindcss/vite";
import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import { playwright } from "@vitest/browser-playwright";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";
import { editionAlias } from "../../packages/edition/node/resolve-entry.ts";
import { fileRoutes } from "./vite/file-routes.ts";
import { routeCatalogs } from "./vite/route-catalogs.ts";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const srcAlias = path.resolve(dirname, "./src");
const sharedAlias = path.resolve(dirname, "../../packages/shared/src");

// Fixtures write the server's moments in UTC; every machine reads them in the same zone.
process.env.TZ = "UTC";

export default defineConfig({
  resolve: {
    alias: { "@": srcAlias, "@trenova/shared": sharedAlias, ...editionAlias() },
    dedupe: ["@tanstack/react-router", "@tanstack/router-core"],
  },
  test: {
    projects: [
      {
        plugins: [fileRoutes({ srcDir: srcAlias }), routeCatalogs({ srcDir: srcAlias })],
        // The setup loads rules.wasm through @trenova/shared/rules/test-module (?inline).
        assetsInclude: ["**/*.wasm"],
        resolve: {
          alias: { "@": srcAlias, "@trenova/shared": sharedAlias, ...editionAlias() },
          dedupe: ["@tanstack/react-router", "@tanstack/router-core"],
        },
        test: {
          name: "unit",
          environment: "happy-dom",
          setupFiles: ["./src/test-setup.ts"],
          include: ["src/**/*.{test,spec}.{ts,tsx}"],
          exclude: ["src/**/*.stories.{ts,tsx}", "src/**/*.compiled.test.{ts,tsx}"],
        },
      },
      {
        // Components built by the React Compiler exactly as vite.config.ts builds the app,
        // for behaviour that only memoization can break, such as a value derived from an
        // object a library changes in place.
        plugins: [
          fileRoutes({ srcDir: srcAlias }),
          routeCatalogs({ srcDir: srcAlias }),
          react({
            compiler: true,
            exclude: [
              /[\\/]node_modules[\\/]/,
              /[\\/]packages[\\/]graphql[\\/]src[\\/]generated[\\/]/,
            ],
          }),
        ],
        assetsInclude: ["**/*.wasm"],
        resolve: {
          alias: { "@": srcAlias, "@trenova/shared": sharedAlias, ...editionAlias() },
          dedupe: ["@tanstack/react-router", "@tanstack/router-core"],
        },
        test: {
          name: "compiled",
          environment: "happy-dom",
          setupFiles: ["./src/test-setup.ts"],
          include: ["src/**/*.compiled.test.{ts,tsx}"],
        },
      },
      {
        plugins: [
          react(),
          tailwindcss(),
          storybookTest({
            configDir: path.join(dirname, ".storybook"),
            storybookScript: "bun run storybook --no-open",
          }),
        ],
        resolve: {
          alias: { "@": srcAlias, "@trenova/shared": sharedAlias, ...editionAlias() },
        },
        optimizeDeps: {
          include: [
            "react",
            "react-dom",
            "react/jsx-dev-runtime",
            "storybook/test",
            "@base-ui/react/collapsible",
            "@tanstack/react-hotkeys",
            "@vis.gl/react-google-maps",
            "react-error-boundary",
            "react-easy-crop",
            "react-lazy-load-image-component",
            "@tanstack/react-router",
            "zustand",
            "zustand/middleware",
          ],
        },
        test: {
          name: "storybook",
          browser: {
            enabled: true,
            provider: playwright({}),
            headless: true,
            instances: [{ browser: "chromium" }],
          },
          setupFiles: ["./.storybook/vitest.setup.ts"],
        },
      },
    ],
  },
});
