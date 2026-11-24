// First, so the design system's stylesheet, and with it Tailwind's cascade layer order,
// comes before any component's CSS: the router's modules bring stylesheets with @layer
// rules of their own, and a layer is ranked by where its name first appears.
import "@trenova/shared/styles/app.css";
import { LazyMotion, domMax } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import { installChunkRecovery } from "@/lib/chunk-recovery";
import { queryClient } from "@/lib/query-client";
import { registerPhotoRefresh } from "@trenova/shared/lib/photo-refresh";
import { predictLinksWhenIdle } from "@/lib/link-foresight";
import { registerServiceWorker } from "@/lib/service-worker";
import { preloadRulesWhenIdle } from "@trenova/shared/rules/preload";
import { Providers } from "./providers";
import { router } from "./router";

installChunkRecovery();
// A picture whose signed link has expired reads the data that carried it again.
registerPhotoRefresh(queryClient);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LazyMotion features={domMax}>
      <Providers>
        <RouterProvider router={router} />
      </Providers>
    </LazyMotion>
  </StrictMode>,
);

registerServiceWorker(router);

// After the page has loaded and the browser is idle, so the formula editor and the rate
// screens find the rules module ready; never before first paint.
preloadRulesWhenIdle();

// Starts a page (code and data) while the pointer is still on its way to its link; records and
// everything else start on hover or focus, the router's intent preload.
predictLinksWhenIdle(router);
