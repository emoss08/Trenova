import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { createRequire } from "node:module";
import path from "path";
import { defineConfig, normalizePath } from "vite";
import { compression } from "vite-plugin-compression2";
import { viteStaticCopy } from "vite-plugin-static-copy";

const require = createRequire(import.meta.url);

const pdfjsDistPath = path.dirname(require.resolve("pdfjs-dist/package.json"));
const cMapsDir = normalizePath(path.join(pdfjsDistPath, "cmaps"));

export default defineConfig({
  plugins: [
    react(),
    // babel({ presets: [reactCompilerPreset()] }),
    tailwindcss(),
    compression({
      algorithms: ["gzip", "brotliCompress"],
      threshold: 10240,
    }),
    viteStaticCopy({
      targets: [
        {
          src: cMapsDir,
          dest: "",
        },
      ],
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
    hmr: {
      timeout: 30000,
      overlay: true,
    },
    warmup: {
      clientFiles: ["./src/lib/utils.ts", "./src/lib/api.ts", "./src/services/api.ts"],
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
            { name: "forms", test: /react-hook-form/ },
            { name: "date-utils", test: /date-fns/ },
            { name: "codemirror-view", test: /@codemirror\/view/ },
            { name: "codemirror-language", test: /@codemirror\/language/ },
            {
              name: "codemirror-autocomplete",
              test: /@codemirror\/autocomplete/,
            },
            { name: "tanstack-vendored", test: /@tanstack/ },
            { name: "base-ui", test: /@base-ui/ },
            { name: "ably", test: /ably/ },
            { name: "framer-motion", test: /motion/ },
            { name: "nivo", test: /@nivo/ },
            { name: "recharts", test: /recharts/ },
            { name: "zod", test: /zod/ },
            { name: "lodash", test: /lodash/ },
            { name: "toast", test: /sonner/ },
            { name: "icons", test: /lucide-react/ },
            {
              name: "phone-utils",
              test(id) {
                return id.includes("react-phone-number-input") || id.includes("libphonenumber-js");
              },
            },
            { name: "country-flags", test: /country-flag-icons/ },
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
