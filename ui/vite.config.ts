/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { nodeResolve } from "@rollup/plugin-node-resolve";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { createRequire } from "node:module";
import path from "path";
import { visualizer } from "rollup-plugin-visualizer";
import tsconfigPaths from "vite-tsconfig-paths";

import { defineConfig, normalizePath, type PluginOption } from "vite";
import { compression } from "vite-plugin-compression2";
import { VitePWA } from "vite-plugin-pwa";
import { viteStaticCopy } from "vite-plugin-static-copy";

const require = createRequire(import.meta.url);
const cMapsDir = normalizePath(
  path.join(path.dirname(require.resolve("pdfjs-dist/package.json")), "cmaps"),
);
const standardFontsDir = normalizePath(
  path.join(
    path.dirname(require.resolve("pdfjs-dist/package.json")),
    "standard_fonts",
  ),
);

const ReactCompilerConfig = {
  /* ... */
};

export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [["babel-plugin-react-compiler", ReactCompilerConfig]],
      },
    }),
    tsconfigPaths(),
    tailwindcss(),
    nodeResolve() as PluginOption,
    VitePWA({
      registerType: "autoUpdate",
      devOptions: {
        enabled: false,
        navigateFallback: "index.html",
        suppressWarnings: true,
        type: "module",
      },
      workbox: {
        globPatterns: ["**/*.{js,css,html,svg,png,ico,webp}"],
        cleanupOutdatedCaches: true,
        clientsClaim: true,
        maximumFileSizeToCacheInBytes: 4 * 1024 * 1024, // 4MB
      },
      pwaAssets: {
        disabled: false,
        config: true,
      },
      manifest: {
        name: "Trenova TMS",
        short_name: "Trenova",
        description:
          "An Open Source AI-driven asset based transportation management system",
        theme_color: "#000000",
      },
    }),
    viteStaticCopy({
      targets: [
        { src: cMapsDir, dest: "" },
        { src: standardFontsDir, dest: "" },
      ],
    }),
    compression({
      algorithms: ["brotliCompress", "gzip"],
      threshold: 512,
      deleteOriginalAssets: false,
    }),
    visualizer({
      open: true,
      gzipSize: true,
      brotliSize: true,
    }) as PluginOption,
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    target: ["es2020", "edge88", "firefox78", "chrome87", "safari14"],
    sourcemap: true,
    reportCompressedSize: true,
    chunkSizeWarningLimit: 1000, // Increase warning limit for chunks
    rollupOptions: {
      output: {
        manualChunks: (id: string) => {
          // Separate heavy libraries that are lazy-loaded
          if (id.includes("recharts")) {
            return "charts";
          }
          if (id.includes("react-pdf") || id.includes("pdfjs-dist")) {
            return "pdf";
          }
          if (id.includes("@tiptap") || id.includes("prosemirror")) {
            return "editor";
          }
          if (
            id.includes("@vis.gl/react-google-maps") ||
            id.includes("@googlemaps")
          ) {
            return "maps";
          }

          // Split vendor libraries into logical groups
          if (id.includes("node_modules")) {
            // Core React libraries
            if (
              id.includes("react") &&
              !id.includes("react-hook-form") &&
              !id.includes("@tanstack")
            ) {
              return "vendor-react";
            }
            // UI libraries (Radix, Ark, etc)
            if (id.includes("@radix-ui") || id.includes("@dnd-kit")) {
              return "vendor-ui";
            }
            // Data management (TanStack, forms, etc)
            if (
              id.includes("@tanstack") ||
              id.includes("react-hook-form") ||
              id.includes("zod") ||
              id.includes("zustand")
            ) {
              return "vendor-data";
            }
            // Icons and assets
            if (id.includes("lucide") || id.includes("@fortawesome")) {
              return "vendor-icons";
            }
            // Everything else
            return "vendor-utils";
          }
        },
        chunkFileNames: (chunkInfo: any) => {
          const facadeModuleId = chunkInfo.facadeModuleId
            ? chunkInfo.facadeModuleId
            : "";
          if (facadeModuleId.includes("node_modules")) {
            return "assets/js/[name]-[hash].js";
          }
          return "assets/js/[name]-[hash].js";
        },
        minifyInternalExports: true,
        assetFileNames: (assetInfo: any) => {
          const info = assetInfo.name?.split(".");
          const extType = info?.[info.length - 1];
          if (extType && /png|jpe?g|svg|webp|gif|tiff|bmp|ico/i.test(extType)) {
            return "assets/images/[name]-[hash][extname]";
          }

          if (extType && /css/i.test(extType)) {
            return "assets/css/[name]-[hash][extname]";
          }

          if (extType && /woff2?|ttf|eot/i.test(extType)) {
            return "assets/fonts/[name]-[hash][extname]";
          }

          return "assets/[ext]/[name]-[hash][extname]";
        },
      },
      input: {
        main: path.resolve(__dirname, "index.html"),
      },
    },
  },

  optimizeDeps: {
    include: ["@tanstack/react-query", "zustand"],
    exclude: ["@vite/client", "@vite/env"],
  },
  server: {
    fs: {
      strict: true,
    },
    hmr: {
      overlay: true,
    },
    warmup: {
      clientFiles: [
        "./src/lib/utils.ts",
        "./src/lib/http-client.ts",
        "./src/lib/json-viewer-utils.ts",
        "./src/services/api.ts",
      ],
    },
  },
});
