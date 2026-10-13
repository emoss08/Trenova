// First, so the design system's stylesheet, and with it Tailwind's cascade layer order,
// comes before any component's CSS: the router's modules bring stylesheets with @layer
// rules of their own, and a layer is ranked by where its name first appears.
import "@trenova/shared/styles/app.css";
import { LazyMotion, domMax } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router";
import { installChunkRecovery } from "@/lib/chunk-recovery";
import { registerServiceWorker } from "@/lib/service-worker";
import { Providers } from "./providers";
import { router } from "./router";

installChunkRecovery();

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
