import tailwindcss from "@tailwindcss/vite";
import { createRequire } from "node:module";
import path from "path";
// import { visualizer } from "rollup-plugin-visualizer";
import { cloudflare } from "@cloudflare/vite-plugin";
import react from "@vitejs/plugin-react";
import { defineConfig, normalizePath } from "vite";
import { compression } from "vite-plugin-compression2";
import { viteStaticCopy } from "vite-plugin-static-copy";

const require = createRequire(import.meta.url);

const pdfjsDistPath = path.dirname(require.resolve("pdfjs-dist/package.json"));
const cMapsDir = normalizePath(path.join(pdfjsDistPath, "cmaps"));
const dirname = import.meta.dirname;

export default defineConfig({
  envDir: path.resolve(dirname, "../.."),
  environments: {
    client: {
      build: {
        rollupOptions: {
          input: {
            main: path.resolve(dirname, "index.html"),
          },
        },
      },
    },
  },
  plugins: [
    react(),
    // babel({ presets: [reactCompilerPreset()] }),
    tailwindcss(),
    compression({
      algorithms: ["gzip", "brotliCompress"],
      threshold: 10240,
    }), // visualizer({
    //   open: !process.env.CI,
    //   gzipSize: true,
    //   brotliSize: true,
    // }),
    viteStaticCopy({
      targets: [
        {
          src: cMapsDir,
          dest: "",
        },
      ],
    }),
    cloudflare(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(dirname, "./src"),
      "@trenova/shared": path.resolve(dirname, "../../packages/shared/src"),
    },
  },
  optimizeDeps: {
    // @foony/realtime builds its Node-only `ws` fallback specifier at runtime
    // (with @vite-ignore) so browser bundlers skip it. Excluding it from
    // pre-bundling keeps esbuild from trying to resolve `ws`.
    exclude: ["@foony/realtime"],
  },
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.setHeader("accept-encoding", "identity");
          });
        },
      },
      "/graphql": {
        target: "http://localhost:8080",
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.setHeader("accept-encoding", "identity");
          });
        },
      },
    },
    hmr: {
      timeout: 30000,
      overlay: true,
    },
    warmup: {
      clientFiles: [
        "../../packages/shared/src/lib/utils.ts",
        "../../packages/shared/src/lib/api.ts",
        "./src/services/api.ts",
      ],
    },
  },
  build: {
    minify: "oxc",
    sourcemap: process.env.NODE_ENV === "development",
    rolldownOptions: {
      preserveEntrySignatures: "strict",
      output: {
        codeSplitting: {
          groups: [
            // Tiny styling helpers that virtually every module reaches through
            // `cn()`. Without a group of their own they get folded into
            // whichever vendor chunk claims them first — recharts did, which
            // dragged recharts + nivo into the entry graph. Keep them first so
            // no other group can swallow them.
            {
              name: "class-utils",
              test(id) {
                return /node_modules[\\/](clsx|tailwind-merge|class-variance-authority|tailwind-variants)[\\/]/.test(
                  id,
                );
              },
            },
            // The framework core is on every page, so it must own a chunk.
            // Left ungrouped it gets absorbed by whatever vendor group claims
            // it first (react-dom landed in the table chunk, @floating-ui in
            // the tiptap chunk), and the entry then pulls that whole library in.
            {
              name: "react-vendor",
              test(id) {
                return /node_modules[\\/](react|react-dom|react-is|scheduler|use-sync-external-store)[\\/]/.test(
                  id,
                );
              },
            },
            { name: "react-router", test: /node_modules[\\/]react-router[\\/]/ },
            { name: "floating-ui", test: /node_modules[\\/]@floating-ui[\\/]/ },
            {
              name: "store-utils",
              test(id) {
                return /node_modules[\\/](zustand|nuqs|immer)[\\/]/.test(id);
              },
            },
            { name: "forms", test: /react-hook-form/ },
            { name: "date-utils", test: /date-fns/ },
            // Shared by @codemirror/view and prosemirror-view. Ungrouped they
            // land in the CodeMirror chunk, which then makes opening the
            // comment editor download all of CodeMirror as well.
            {
              name: "editor-dom-utils",
              test: /node_modules[\\/](w3c-keyname|style-mod|crelt)[\\/]/,
            },
            { name: "codemirror-view", test: /@codemirror\/view/ },
            { name: "codemirror-language", test: /@codemirror\/language/ },
            {
              name: "codemirror-autocomplete",
              test: /@codemirror\/autocomplete/,
            },
            // Table/virtualiser code only ever runs behind a lazy data table;
            // keep it out of the react-query chunk the entry needs.
            {
              name: "tanstack-table",
              test(id) {
                return (
                  id.includes("@tanstack/react-table") ||
                  id.includes("@tanstack/table-core") ||
                  id.includes("@tanstack/react-virtual") ||
                  id.includes("@tanstack/virtual-core")
                );
              },
            },
            { name: "tanstack-vendored", test: /@tanstack/ },
            {
              name: "tiptap",
              test(id) {
                return id.includes("@tiptap") || id.includes("prosemirror");
              },
            },
            { name: "dnd-kit", test: /@dnd-kit/ },
            { name: "cmdk", test: /node_modules[\\/]cmdk[\\/]/ },
            { name: "google-maps", test: /@vis\.gl[\\/]react-google-maps/ },
            { name: "base-ui", test: /@base-ui/ },
            { name: "foony", test: /@foony/ },
            { name: "framer-motion", test: /motion/ },
            { name: "nivo", test: /@nivo/ },
            { name: "recharts", test: /recharts/ },
            { name: "zod", test: /zod/ },
            { name: "lodash", test: /lodash/ },
            { name: "toast", test: /sonner/ },
            { name: "icons", test: /lucide-react/ },
            { name: "pdfjs", test: /pdfjs-dist/ },
            // Must precede phone-utils: `react-phone-number-input/flags` is
            // the barrel over the flag components and is imported lazily, so it
            // belongs with the flags rather than with the phone metadata the
            // input needs up front. Scoped to the React components on purpose —
            // react-phone-number-input reaches for country-flag-icons' tiny
            // emoji helper synchronously, and grouping that with the SVGs would
            // pull all 232 kB of them back into the eager graph.
            {
              name: "country-flags",
              test(id) {
                return (
                  /node_modules[\\/]country-flag-icons[\\/]react[\\/]/.test(id) ||
                  /react-phone-number-input[\\/]flags/.test(id)
                );
              },
            },
            {
              name: "phone-utils",
              test(id) {
                return id.includes("react-phone-number-input") || id.includes("libphonenumber-js");
              },
            },
            {
              name: "http-client",
              test(id) {
                return id.includes("axios") || id.includes("fetch");
              },
            },
          ],
        },
        entryFileNames: "assets/js/[name].[hash].js",
        chunkFileNames: "assets/js/[name].[hash].js",
        assetFileNames: (assetInfo) => {
          if (assetInfo.names?.[0] && /\.(woff|woff2|eot|ttf|otf)$/.test(assetInfo.names[0])) {
            return "assets/fonts/[name][extname]";
          }
          return "assets/[name].[hash][extname]";
        },
      },
    },
  },
});
