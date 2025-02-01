import { nodeResolve } from "@rollup/plugin-node-resolve";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "path";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig, type PluginOption } from "vite";
import { compression } from "vite-plugin-compression2";
import { VitePWA } from "vite-plugin-pwa";

const ReactCompilerConfig = {
  target: "18", // '17' | '18' | '19'
};

// Define vendor chunks that should be bundled separately
const vendorChunks = {
  // UI Framework and Core
  "core-react": ["react", "react-dom", "react-router-dom"],

  // State Management and Data Fetching
  "data-management": ["@tanstack/react-query", "zustand"],

  // UI Components and Styling
  "ui-components": [
    "@radix-ui/react-alert-dialog",
    "@radix-ui/react-avatar",
    "@radix-ui/react-checkbox",
    "@radix-ui/react-collapsible",
    "@radix-ui/react-dialog",
    "@radix-ui/react-dropdown-menu",
    "@radix-ui/react-label",
    "@radix-ui/react-popover",
    "@radix-ui/react-radio-group",
    "@radix-ui/react-scroll-area",
    "@radix-ui/react-select",
    "@radix-ui/react-slot",
    "@radix-ui/react-tooltip",
    "@radix-ui/react-visually-hidden",
  ],

  // Table and Query functionality
  "data-tables": ["@tanstack/react-table"],

  // Form Management
  "form-handling": ["react-hook-form", "@hookform/resolvers", "zod", "yup"],

  // Drag and Drop
  "dnd-kit": [
    "@dnd-kit/core",
    "@dnd-kit/modifiers",
    "@dnd-kit/sortable",
    "@dnd-kit/utilities",
  ],

  // Icons and Assets
  icons: [
    "@fortawesome/pro-regular-svg-icons",
    "@fortawesome/pro-solid-svg-icons",
  ],

  // Maps
  "google-maps": ["@vis.gl/react-google-maps"],

  // Date handling
  "date-utils": ["date-fns", "chrono-node"],

  // Utilities
  utils: ["clsx", "tailwind-merge", "class-variance-authority"],
};

export default defineConfig(({ mode }) => ({
  plugins: [
    react({
      babel: {
        plugins: [["babel-plugin-react-compiler", ReactCompilerConfig]],
      },
    }),
    tailwindcss(),
    nodeResolve(),
    VitePWA({
      registerType: "autoUpdate",
      devOptions: {
        enabled: true,
      },
      includeAssets: [
        "favicon.ico",
        "logo.webp",
        "apple-touch-icon.png",
        "mask-icon.svg",
      ],
      manifest: {
        name: "Trenova TMS",
        short_name: "Trenova",
        description:
          "An Open Source AI-driven asset based transportation management system",
        theme_color: "#000000",
        icons: [
          {
            src: "/favicon.ico",
            sizes: "any",
            type: "image/x-icon",
          },
          {
            src: "/logo.webp",
            sizes: "any",
            type: "image/webp",
          },
        ],
      },
    }),
    compression({
      // Add compression for production builds
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
      template: "treemap", // Use treemap for better visualization
    }) as PluginOption,
  ],

  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },

  build: {
    target: ["es2020", "edge88", "firefox78", "chrome87", "safari14"],
    sourcemap: mode === "development",
    reportCompressedSize: true,
    chunkSizeWarningLimit: 1000, // Increase warning limit for chunks
    rollupOptions: {
      output: {
        manualChunks: {
          ...vendorChunks,
          // Dynamic chunks for routes
          ...(() => {
            const dynamicImports = {};
            // Add dynamic imports for each major feature
            return dynamicImports;
          })(),
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
        minifyInternalExports: true,
        assetFileNames: (assetInfo) => {
          const info = assetInfo.name.split(".");
          const extType = info[info.length - 1];
          if (/png|jpe?g|svg|gif|tiff|bmp|ico/i.test(extType)) {
            return "assets/images/[name]-[hash][extname]";
          }
          if (/css/i.test(extType)) {
            return "assets/css/[name]-[hash][extname]";
          }
          if (/woff2?|ttf|eot/i.test(extType)) {
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
  },
}));
