import tailwindcss from "@tailwindcss/vite";
import { createRequire } from "node:module";
import path from "path";
// import { visualizer } from "rollup-plugin-visualizer";
import react from "@vitejs/plugin-react";
import { defineConfig, normalizePath } from "vite";
import { viteStaticCopy } from "vite-plugin-static-copy";
import { editionAlias } from "../../packages/edition/node/resolve-entry.ts";
import { singletonPackages } from "../../singleton-packages.ts";
import { captureDiscovery } from "./vite/capture-discovery.ts";
import { chunkGroups } from "./vite/chunk-groups.ts";
import { routeCatalogs } from "./vite/route-catalogs.ts";
import { serviceWorker } from "./vite/service-worker.ts";

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
    routeCatalogs({ srcDir: path.resolve(dirname, "src") }),
    captureDiscovery(),
    // Every component is compiled, by the React Compiler's Rust port in oxc-transform-react
    // rather than the Babel plugin, which held the main thread for nine-tenths of a build.
    // Files built on TanStack Virtual, whose virtualizer changes in place, start with
    // "use no memo", as does any page found showing stale values: that is how a page opts
    // out. TanStack Table v9 is compiled; a component that reads table state through a
    // row, column or header subscribes to it. The generated GraphQL client is megabytes of
    // documents and types with no components, so it is left out.
    react({
      compiler: true,
      exclude: [/[\\/]node_modules[\\/]/, /[\\/]packages[\\/]graphql[\\/]src[\\/]generated[\\/]/],
    }),
    tailwindcss(),
    // visualizer({
    //   open: !process.env.CI,
    //   gzipSize: true,
    //   brotliSize: true,
    // }),
    serviceWorker({ source: path.resolve(dirname, "src/service-worker/sw.ts") }),
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
      "@": path.resolve(dirname, "./src"),
      "@trenova/shared": path.resolve(dirname, "../../packages/shared/src"),
      ...editionAlias(),
    },
    // Shared components import these from packages/shared. A second copy resolved there
    // carries its own React context: router hooks report they are outside a router, a
    // shared DialogTitle throws Base UI error #27. See singleton-packages.ts.
    dedupe: [...singletonPackages],
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
      clientFiles: ["./src/main.tsx"],
    },
  },
  build: {
    minify: "oxc",
    sourcemap: process.env.NODE_ENV === "development",
    rolldownOptions: {
      // Leaves the unused re-exports of side-effect-free barrel modules in dependencies
      // unparsed: about a tenth of the modules a build used to load. The bundle is unchanged.
      experimental: { lazyBarrel: true },
      preserveEntrySignatures: "strict",
      output: {
        codeSplitting: {
          groups: chunkGroups,
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
