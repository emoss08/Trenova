import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vite";
import { compression } from "vite-plugin-compression2";

const proxyConfig = {
  target: "http://localhost:8080",
  changeOrigin: true,
  configure(proxy: { on: (e: string, cb: (req: { setHeader: (k: string, v: string) => void }) => void) => void }) {
    proxy.on("proxyReq", (proxyReq) => {
      proxyReq.setHeader("accept-encoding", "identity");
    });
  },
};

export default defineConfig({
  envDir: path.resolve(__dirname, "../.."),
  plugins: [
    react(),
    tailwindcss(),
    compression({
      algorithms: ["gzip", "brotliCompress"],
      threshold: 10240,
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
      "@trenova/shared": path.resolve(__dirname, "../../packages/shared/src"),
    },
  },
  optimizeDeps: {
    // @foony/realtime builds its Node-only `ws` fallback specifier at runtime
    // (with @vite-ignore) so browser bundlers skip it. Excluding it from
    // pre-bundling keeps esbuild from trying to resolve `ws`.
    exclude: ["@foony/realtime"],
  },
  server: {
    port: 5174,
    strictPort: true,
    proxy: {
      "/api": proxyConfig,
      "/graphql": proxyConfig,
    },
  },
  build: {
    minify: "oxc",
    sourcemap: process.env.NODE_ENV === "development",
  },
});
