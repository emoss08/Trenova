/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
import react from "@vitejs/plugin-react-swc";
import path from "path";
import { visualizer } from "rollup-plugin-visualizer";
import { defineConfig, loadEnv } from "vite";
import { VitePWA } from "vite-plugin-pwa";
export default defineConfig(function (_a) {
    var mode = _a.mode;
    // Load env file based on `mode` in the current working directory.
    // Set the third parameter to '' to load all env regardless of the `VITE_` prefix.
    var env = loadEnv(mode, process.cwd(), "");
    return {
        test: {
            css: false,
            environment: "jsdom",
            include: ["src/**/__tests__/*"],
            globals: true,
            clearMocks: true,
        },
        plugins: __spreadArray([
            react(),
            visualizer({
                open: true,
                gzipSize: true,
                brotliSize: true,
            })
        ], (mode === "test"
            ? []
            : [
                VitePWA({
                    registerType: "autoUpdate",
                }),
            ]), true),
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
                    manualChunks: function (id) {
                        if (id.includes("node_modules")) {
                            var packages = [
                                "lodash",
                                "date-fns",
                                "react-hook-form",
                                "@radix-ui",
                                "react-aria",
                                "react-beautiful-dnd",
                                "i18next",
                                "@fortawesome",
                            ];
                            var chunk = packages.find(function (pkg) {
                                return id.includes("/node_modules/".concat(pkg));
                            });
                            return chunk ? "vendor.".concat(chunk) : null;
                        }
                    },
                },
            },
        },
    };
});
