// vite.config.ts
import react from "file:///home/wolfred/Desktop/Trenova/client/node_modules/@vitejs/plugin-react-swc/index.mjs";
import million from "file:///home/wolfred/Desktop/Trenova/client/node_modules/million/dist/packages/compiler.mjs";
import path from "path";
import { visualizer } from "file:///home/wolfred/Desktop/Trenova/client/node_modules/rollup-plugin-visualizer/dist/plugin/index.js";
import { defineConfig, loadEnv } from "file:///home/wolfred/Desktop/Trenova/client/node_modules/vite/dist/node/index.js";
import { VitePWA } from "file:///home/wolfred/Desktop/Trenova/client/node_modules/vite-plugin-pwa/dist/index.js";
var __vite_injected_original_dirname = "/home/wolfred/Desktop/Trenova/client";
var vite_config_default = defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  return {
    test: {
      css: false,
      environment: "jsdom",
      include: ["src/**/__tests__/*"],
      globals: true,
      clearMocks: true
    },
    plugins: [
      million.vite({ auto: true }),
      react(),
      visualizer({
        open: true,
        gzipSize: true,
        brotliSize: true
      }),
      ...mode === "test" ? [] : [
        VitePWA({
          registerType: "autoUpdate"
        })
      ]
    ],
    resolve: {
      alias: {
        "@": path.resolve(__vite_injected_original_dirname, "./src")
      }
    },
    define: {
      __APP_ENV__: JSON.stringify(env.APP_ENV)
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
                "@fortawesome"
              ];
              const chunk = packages.find(
                (pkg) => id.includes(`/node_modules/${pkg}`)
              );
              return chunk ? `vendor.${chunk}` : null;
            }
          }
        }
      }
    }
  };
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcudHMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvaG9tZS93b2xmcmVkL0Rlc2t0b3AvVHJlbm92YS9jbGllbnRcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9ob21lL3dvbGZyZWQvRGVza3RvcC9UcmVub3ZhL2NsaWVudC92aXRlLmNvbmZpZy50c1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vaG9tZS93b2xmcmVkL0Rlc2t0b3AvVHJlbm92YS9jbGllbnQvdml0ZS5jb25maWcudHNcIjtpbXBvcnQgcmVhY3QgZnJvbSBcIkB2aXRlanMvcGx1Z2luLXJlYWN0LXN3Y1wiO1xyXG5pbXBvcnQgbWlsbGlvbiBmcm9tIFwibWlsbGlvbi9jb21waWxlclwiO1xyXG5pbXBvcnQgcGF0aCBmcm9tIFwicGF0aFwiO1xyXG5pbXBvcnQgeyB2aXN1YWxpemVyIH0gZnJvbSBcInJvbGx1cC1wbHVnaW4tdmlzdWFsaXplclwiO1xyXG5pbXBvcnQgeyBkZWZpbmVDb25maWcsIGxvYWRFbnYsIHR5cGUgUGx1Z2luT3B0aW9uIH0gZnJvbSBcInZpdGVcIjtcclxuaW1wb3J0IHsgVml0ZVBXQSB9IGZyb20gXCJ2aXRlLXBsdWdpbi1wd2FcIjtcclxuXHJcbmV4cG9ydCBkZWZhdWx0IGRlZmluZUNvbmZpZygoeyBtb2RlIH0pID0+IHtcclxuICAvLyBMb2FkIGVudiBmaWxlIGJhc2VkIG9uIGBtb2RlYCBpbiB0aGUgY3VycmVudCB3b3JraW5nIGRpcmVjdG9yeS5cclxuICAvLyBTZXQgdGhlIHRoaXJkIHBhcmFtZXRlciB0byAnJyB0byBsb2FkIGFsbCBlbnYgcmVnYXJkbGVzcyBvZiB0aGUgYFZJVEVfYCBwcmVmaXguXHJcbiAgY29uc3QgZW52ID0gbG9hZEVudihtb2RlLCBwcm9jZXNzLmN3ZCgpLCBcIlwiKTtcclxuXHJcbiAgcmV0dXJuIHtcclxuICAgIHRlc3Q6IHtcclxuICAgICAgY3NzOiBmYWxzZSxcclxuICAgICAgZW52aXJvbm1lbnQ6IFwianNkb21cIixcclxuICAgICAgaW5jbHVkZTogW1wic3JjLyoqL19fdGVzdHNfXy8qXCJdLFxyXG4gICAgICBnbG9iYWxzOiB0cnVlLFxyXG4gICAgICBjbGVhck1vY2tzOiB0cnVlLFxyXG4gICAgfSxcclxuICAgIHBsdWdpbnM6IFtcclxuICAgICAgbWlsbGlvbi52aXRlKHsgYXV0bzogdHJ1ZSB9KSxcclxuICAgICAgcmVhY3QoKSxcclxuICAgICAgdmlzdWFsaXplcih7XHJcbiAgICAgICAgb3BlbjogdHJ1ZSxcclxuICAgICAgICBnemlwU2l6ZTogdHJ1ZSxcclxuICAgICAgICBicm90bGlTaXplOiB0cnVlLFxyXG4gICAgICB9KSBhcyBQbHVnaW5PcHRpb24sXHJcbiAgICAgIC4uLihtb2RlID09PSBcInRlc3RcIlxyXG4gICAgICAgID8gW11cclxuICAgICAgICA6IFtcclxuICAgICAgICAgICAgVml0ZVBXQSh7XHJcbiAgICAgICAgICAgICAgcmVnaXN0ZXJUeXBlOiBcImF1dG9VcGRhdGVcIixcclxuICAgICAgICAgICAgfSksXHJcbiAgICAgICAgICBdKSxcclxuICAgIF0sXHJcbiAgICByZXNvbHZlOiB7XHJcbiAgICAgIGFsaWFzOiB7XHJcbiAgICAgICAgXCJAXCI6IHBhdGgucmVzb2x2ZShfX2Rpcm5hbWUsIFwiLi9zcmNcIiksXHJcbiAgICAgIH0sXHJcbiAgICB9LFxyXG4gICAgZGVmaW5lOiB7XHJcbiAgICAgIF9fQVBQX0VOVl9fOiBKU09OLnN0cmluZ2lmeShlbnYuQVBQX0VOViksXHJcbiAgICB9LFxyXG4gICAgYnVpbGQ6IHtcclxuICAgICAgcm9sbHVwT3B0aW9uczoge1xyXG4gICAgICAgIG91dHB1dDoge1xyXG4gICAgICAgICAgbWFudWFsQ2h1bmtzKGlkKSB7XHJcbiAgICAgICAgICAgIGlmIChpZC5pbmNsdWRlcyhcIm5vZGVfbW9kdWxlc1wiKSkge1xyXG4gICAgICAgICAgICAgIGNvbnN0IHBhY2thZ2VzID0gW1xyXG4gICAgICAgICAgICAgICAgXCJsb2Rhc2hcIixcclxuICAgICAgICAgICAgICAgIFwiZGF0ZS1mbnNcIixcclxuICAgICAgICAgICAgICAgIFwicmVhY3QtaG9vay1mb3JtXCIsXHJcbiAgICAgICAgICAgICAgICBcIkByYWRpeC11aVwiLFxyXG4gICAgICAgICAgICAgICAgXCJyZWFjdC1hcmlhXCIsXHJcbiAgICAgICAgICAgICAgICBcInJlYWN0LWRvbVwiLFxyXG4gICAgICAgICAgICAgICAgXCJAZm9ydGF3ZXNvbWVcIixcclxuICAgICAgICAgICAgICBdO1xyXG4gICAgICAgICAgICAgIGNvbnN0IGNodW5rID0gcGFja2FnZXMuZmluZCgocGtnKSA9PlxyXG4gICAgICAgICAgICAgICAgaWQuaW5jbHVkZXMoYC9ub2RlX21vZHVsZXMvJHtwa2d9YCksXHJcbiAgICAgICAgICAgICAgKTtcclxuICAgICAgICAgICAgICByZXR1cm4gY2h1bmsgPyBgdmVuZG9yLiR7Y2h1bmt9YCA6IG51bGw7XHJcbiAgICAgICAgICAgIH1cclxuICAgICAgICAgIH0sXHJcbiAgICAgICAgfSxcclxuICAgICAgfSxcclxuICAgIH0sXHJcbiAgfTtcclxufSk7XHJcbiJdLAogICJtYXBwaW5ncyI6ICI7QUFBOFIsT0FBTyxXQUFXO0FBQ2hULE9BQU8sYUFBYTtBQUNwQixPQUFPLFVBQVU7QUFDakIsU0FBUyxrQkFBa0I7QUFDM0IsU0FBUyxjQUFjLGVBQWtDO0FBQ3pELFNBQVMsZUFBZTtBQUx4QixJQUFNLG1DQUFtQztBQU96QyxJQUFPLHNCQUFRLGFBQWEsQ0FBQyxFQUFFLEtBQUssTUFBTTtBQUd4QyxRQUFNLE1BQU0sUUFBUSxNQUFNLFFBQVEsSUFBSSxHQUFHLEVBQUU7QUFFM0MsU0FBTztBQUFBLElBQ0wsTUFBTTtBQUFBLE1BQ0osS0FBSztBQUFBLE1BQ0wsYUFBYTtBQUFBLE1BQ2IsU0FBUyxDQUFDLG9CQUFvQjtBQUFBLE1BQzlCLFNBQVM7QUFBQSxNQUNULFlBQVk7QUFBQSxJQUNkO0FBQUEsSUFDQSxTQUFTO0FBQUEsTUFDUCxRQUFRLEtBQUssRUFBRSxNQUFNLEtBQUssQ0FBQztBQUFBLE1BQzNCLE1BQU07QUFBQSxNQUNOLFdBQVc7QUFBQSxRQUNULE1BQU07QUFBQSxRQUNOLFVBQVU7QUFBQSxRQUNWLFlBQVk7QUFBQSxNQUNkLENBQUM7QUFBQSxNQUNELEdBQUksU0FBUyxTQUNULENBQUMsSUFDRDtBQUFBLFFBQ0UsUUFBUTtBQUFBLFVBQ04sY0FBYztBQUFBLFFBQ2hCLENBQUM7QUFBQSxNQUNIO0FBQUEsSUFDTjtBQUFBLElBQ0EsU0FBUztBQUFBLE1BQ1AsT0FBTztBQUFBLFFBQ0wsS0FBSyxLQUFLLFFBQVEsa0NBQVcsT0FBTztBQUFBLE1BQ3RDO0FBQUEsSUFDRjtBQUFBLElBQ0EsUUFBUTtBQUFBLE1BQ04sYUFBYSxLQUFLLFVBQVUsSUFBSSxPQUFPO0FBQUEsSUFDekM7QUFBQSxJQUNBLE9BQU87QUFBQSxNQUNMLGVBQWU7QUFBQSxRQUNiLFFBQVE7QUFBQSxVQUNOLGFBQWEsSUFBSTtBQUNmLGdCQUFJLEdBQUcsU0FBUyxjQUFjLEdBQUc7QUFDL0Isb0JBQU0sV0FBVztBQUFBLGdCQUNmO0FBQUEsZ0JBQ0E7QUFBQSxnQkFDQTtBQUFBLGdCQUNBO0FBQUEsZ0JBQ0E7QUFBQSxnQkFDQTtBQUFBLGdCQUNBO0FBQUEsY0FDRjtBQUNBLG9CQUFNLFFBQVEsU0FBUztBQUFBLGdCQUFLLENBQUMsUUFDM0IsR0FBRyxTQUFTLGlCQUFpQixHQUFHLEVBQUU7QUFBQSxjQUNwQztBQUNBLHFCQUFPLFFBQVEsVUFBVSxLQUFLLEtBQUs7QUFBQSxZQUNyQztBQUFBLFVBQ0Y7QUFBQSxRQUNGO0FBQUEsTUFDRjtBQUFBLElBQ0Y7QUFBQSxFQUNGO0FBQ0YsQ0FBQzsiLAogICJuYW1lcyI6IFtdCn0K
