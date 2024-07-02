import react from "@vitejs/plugin-react-swc";
import path from "path";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig, loadEnv, PluginOption } from "vite";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig(({ mode }) => {
  // Load env file based on `mode` in the current working directory.
  // Set the third parameter to '' to load all env regardless of the `VITE_` prefix.
  const env = loadEnv(mode, process.cwd(), "");

  return {
    test: {
      css: false,
      environment: "jsdom",
      include: ["src/**/__tests__/*"],
      globals: true,
      clearMocks: true,
    },
    plugins: [
      react(),
      visualizer({
        open: true,
        gzipSize: true,
        brotliSize: true,
      }) as unknown as PluginOption,
      ...(mode === "test"
        ? []
        : [
            VitePWA({
              registerType: "autoUpdate",
            }),
          ]),
    ],
    resolve: {
      alias: {
        "@": path.resolve(__dirname, "./src"),
      },
    },
    define: {
      __APP_ENV__: JSON.stringify(env.APP_ENV),
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes("node_modules")) {
              const packages = [
                "lodash",
                "date-fns",
                "react-hook-form",
                "@radix-ui",
                "react-aria",
                "react-dom",
                "@fortawesome",
              ];
              const chunk = packages.find((pkg) =>
                id.includes(`/node_modules/${pkg}`),
              );
              return chunk ? `vendor.${chunk}` : null;
            }
          },
        },
      },
    },
  };
});
