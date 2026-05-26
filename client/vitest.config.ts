import tailwindcss from "@tailwindcss/vite";
import { storybookTest } from "@storybook/addon-vitest/vitest-plugin";
import { playwright } from "@vitest/browser-playwright";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { defineConfig } from "vitest/config";

const dirname = path.dirname(fileURLToPath(import.meta.url));
const srcAlias = path.resolve(dirname, "./src");

export default defineConfig({
  resolve: {
    alias: { "@": srcAlias },
  },
  test: {
    projects: [
      {
        resolve: {
          alias: { "@": srcAlias },
        },
        test: {
          name: "unit",
          environment: "happy-dom",
          setupFiles: ["./src/test-setup.ts"],
          include: ["src/**/*.{test,spec}.{ts,tsx}"],
          exclude: ["src/**/*.stories.{ts,tsx}"],
        },
      },
      {
        plugins: [
          react(),
          tailwindcss(),
          storybookTest({
            configDir: path.join(dirname, ".storybook"),
            storybookScript: "pnpm storybook --no-open",
          }),
        ],
        resolve: {
          alias: { "@": srcAlias },
        },
        optimizeDeps: {
          include: [
            "react",
            "react-dom",
            "react/jsx-dev-runtime",
            "storybook/test",
            "@base-ui/react/avatar",
            "@base-ui/react/collapsible",
            "@tanstack/react-hotkeys",
            "@vis.gl/react-google-maps",
            "nuqs/adapters/react-router/v7",
            "react-error-boundary",
            "react-image-crop",
            "react-lazy-load-image-component",
            "react-router",
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
