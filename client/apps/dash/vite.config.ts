import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import path from "node:path";
import { defineConfig } from "vite";
import { compression } from "vite-plugin-compression2";
import { singletonPackages } from "../../singleton-packages.ts";

const proxyConfig = {
  target: "http://localhost:8080",
  changeOrigin: true,
  configure(proxy: {
    on: (e: string, cb: (req: { setHeader: (k: string, v: string) => void }) => void) => void;
  }) {
    proxy.on("proxyReq", (proxyReq) => {
      proxyReq.setHeader("accept-encoding", "identity");
    });
  },
};

export default defineConfig({
  base: "/dash/",
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
    // Shared components import these from packages/shared. A second copy resolved there
    // carries its own React context: router hooks report they are outside a router, a
    // shared DialogTitle throws Base UI error #27. See singleton-packages.ts.
    dedupe: [...singletonPackages],
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
    outDir: "dist/dash",
    sourcemap: process.env.NODE_ENV === "development",
  },
});
