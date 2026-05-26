import tailwindcss from "@tailwindcss/vite";
import type { StorybookConfig } from "@storybook/react-vite";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { mergeConfig } from "vite";

const dirname = path.dirname(fileURLToPath(import.meta.url));

const config: StorybookConfig = {
  stories: ["../src/**/*.mdx", "../src/**/*.stories.@(ts|tsx)"],
  addons: [
    "@storybook/addon-docs",
    "@storybook/addon-a11y",
    "@storybook/addon-themes",
    "@storybook/addon-vitest",
  ],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  core: {
    builder: {
      name: "@storybook/builder-vite",
      options: {
        viteConfigPath: path.resolve(dirname, "storybook-only.vite.config.ts"),
      },
    },
  },
  docs: {
    autodocs: "tag",
  },
  async viteFinal(baseConfig) {
    return mergeConfig(baseConfig, {
      plugins: [tailwindcss()],
      resolve: {
        alias: {
          "@": path.resolve(dirname, "../src"),
        },
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
    });
  },
};

export default config;
