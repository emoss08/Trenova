import { nodeResolve } from "@rollup/plugin-node-resolve";
// @ts-expect-error // Module does not give types
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { createRequire } from "node:module";
import path from "path";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig, normalizePath, type PluginOption } from "vite";
import { compression } from "vite-plugin-compression2";
import { viteStaticCopy } from "vite-plugin-static-copy";

// Define vendor chunks that should be bundled separately
// const vendorChunks = {
//   // UI Framework and Core
//   "core-react": [
//     "react",
//     "react-dom",
//     "react-router-dom",
//     "react-helmet-async",
//   ],

//   // State Management and Data Fetching
//   "data-management": ["@tanstack/react-query", "zustand"],

//   // UI Components and Styling
//   "ui-components": [
//     "@radix-ui/react-alert-dialog",
//     "@radix-ui/react-avatar",
//     "@radix-ui/react-checkbox",
//     "@radix-ui/react-collapsible",
//     "@radix-ui/react-dialog",
//     "@radix-ui/react-dropdown-menu",
//     "@radix-ui/react-label",
//     "@radix-ui/react-popover",
//     "@radix-ui/react-radio-group",
//     "@radix-ui/react-scroll-area",
//     "@radix-ui/react-select",
//     "@radix-ui/react-slot",
//     "@radix-ui/react-tooltip",
//     "@radix-ui/react-tabs",
//     "@radix-ui/react-visually-hidden",
//     "@radix-ui/react-switch",
//     "react-lazy-load-image-component",
//     "nuqs",
//     "sonner",
//     "react-day-picker",
//     "react-markdown",
//     "@ark-ui/react",
//   ],

//   "pdf-js": ["react-pdf"],

//   // Table and Query functionality
//   "data-tables": ["@tanstack/react-table"],

//   // Form Management
//   "form-handling": ["react-hook-form", "@hookform/resolvers", "zod"],

//   // Drag and Drop
//   "dnd-kit": [
//     "@dnd-kit/core",
//     "@dnd-kit/modifiers",
//     "@dnd-kit/sortable",
//     "@dnd-kit/utilities",
//   ],

//   "sql-editor": ["ace-builds", "react-ace", "sql-formatter"],

//   // Icons and Assets
//   icons: [
//     "@radix-ui/react-icons",
//     "@fortawesome/pro-regular-svg-icons",
//     "@fortawesome/pro-solid-svg-icons",
//   ],

//   // Date handling
//   "date-utils": ["date-fns", "chrono-node"],

//   // Animation
//   animation: ["motion"],

//   // Utilities
//   utils: ["clsx", "tailwind-merge", "class-variance-authority"],
// };

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
const aceBuildsDir = normalizePath(
  path.join(
    path.dirname(require.resolve("ace-builds/package.json")),
    "src-noconflict",
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
    tailwindcss(),
    nodeResolve() as PluginOption,
    // VitePWA({
    //   registerType: "autoUpdate",
    //   devOptions: {
    //     enabled: false,
    //     navigateFallback: "index.html",
    //     suppressWarnings: true,
    //     type: "module",
    //   },
    //   workbox: {
    //     globPatterns: ["**/*.{js,css,html,svg,png,ico,webp}"],
    //     cleanupOutdatedCaches: true,
    //     clientsClaim: true,
    //     maximumFileSizeToCacheInBytes: 4 * 1024 * 1024, // 4MB
    //   },
    //   pwaAssets: {
    //     disabled: false,
    //     config: true,
    //   },
    //   manifest: {
    //     name: "Trenova TMS",
    //     short_name: "Trenova",
    //     description:
    //       "An Open Source AI-driven asset based transportation management system",
    //     theme_color: "#000000",
    //   },
    // }),
    viteStaticCopy({
      targets: [
        { src: cMapsDir, dest: "" },
        { src: standardFontsDir, dest: "" },
        { src: aceBuildsDir, dest: "ace-builds" },
      ],
    }),
    compression({
      algorithm: "brotliCompress",
      threshold: 512,
      deleteOriginalAssets: false,
    }),
    compression({
      algorithm: "gzip",
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
        advancedChunks: {
          minSize: 1000,
          maxSize: 2000,
          maxModuleSize: 1000,
          minModuleSize: 100,
          minShareCount: 2,
          groups: [
            {
              name: "core-react",
              test: /node_modules\/(react|react-dom|react-router-dom|react-helmet-async)\//,
              priority: 10,
            },
            {
              name: "data-management",
              test: /node_modules\/(@tanstack\/react-query|zustand)\//,
              priority: 9,
            },
            {
              name: "ui-components",
              test: /node_modules\/(@radix-ui\/react-|@ark-ui\/react|nuqs|sonner|react-day-picker|react-markdown|react-lazy-load-image-component)\//,
              priority: 8,
            },
            {
              name: "pdf-js",
              test: /node_modules\/react-pdf\//,
              priority: 10,
            },
            {
              name: "data-tables",
              test: /node_modules\/@tanstack\/react-table\//,
              priority: 7,
            },
            {
              name: "form-handling",
              test: /node_modules\/(react-hook-form|@hookform\/resolvers|zod)\//,
              priority: 6,
            },
            {
              name: "dnd-kit",
              test: /node_modules\/@dnd-kit\/(core|modifiers|sortable|utilities)\//,
              priority: 5,
            },
            {
              name: "sql-editor",
              test: /node_modules\/(ace-builds|react-ace|sql-formatter)\//,
              priority: 4,
            },
            {
              name: "icons",
              test: /node_modules\/(@radix-ui\/react-icons|@fortawesome\/(pro-regular-svg-icons|pro-solid-svg-icons))\//,
              priority: 3,
            },
            {
              name: "date-utils",
              test: /node_modules\/(date-fns|chrono-node)\//,
              priority: 2,
            },
            {
              name: "animation",
              test: /node_modules\/(motion|tw-animate-css)\//,
              priority: 1,
            },
            {
              name: "utils",
              test: /node_modules\/(clsx|tailwind-merge|class-variance-authority)\//,
              priority: 0,
            },
          ],
        },
        chunkFileNames: (chunkInfo) => {
          const name = chunkInfo.name || "chunk";
          if (chunkInfo.moduleIds.length > 0) {
            const moduleId = chunkInfo.moduleIds[0];
            if (moduleId.includes("node_modules")) {
              const packageName = moduleId
                .split("node_modules/")[1]
                .split("/")[0]
                .replace("@", "");
              return `assets/js/vendor/${packageName}-[hash].js`;
            }
            if (moduleId.includes("src/")) {
              const match = moduleId.match(/src\/([^/]+)/);
              if (match) {
                return `assets/js/app/${match[1]}-[hash].js`;
              }
            }
          }
          return `assets/js/${name}-[hash].js`;
        },
        // minifyInternalExports: true,
        assetFileNames: (assetInfo) => {
          const info = assetInfo.names[0]?.split(".");
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
