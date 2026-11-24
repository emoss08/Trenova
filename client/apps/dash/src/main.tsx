import { registerPhotoRefresh } from "@trenova/shared/lib/photo-refresh";
import { LazyMotion, domAnimation } from "motion/react";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RootErrorBoundary } from "@trenova/shared/components/error-boundary";
import { ThemeProvider } from "@trenova/shared/components/theme-provider";
import { Toaster } from "@trenova/shared/components/ui/toaster";
import { DriverAppPlanGate } from "./_components/driver-app-plan-gate";
import { installDashPlanLimitHandler } from "./lib/plan-handler";
import { installDashSessionContextHandler } from "./lib/session-context-handler";
import { dashRouter } from "./router";
import "@trenova/shared/styles/app.css";

// A sign-out, sign-in or organization switch in another tab ends what this tab shows.
installDashSessionContextHandler();
// A company whose plan does not include the driver app gets one plain message, not errors.
installDashPlanLimitHandler();

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      staleTime: 30 * 1000,
      gcTime: 10 * 60 * 1000,
    },
  },
});

// A picture whose signed link has expired reads the data that carried it again.
registerPhotoRefresh(queryClient);

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <LazyMotion features={domAnimation}>
      <RootErrorBoundary>
        <QueryClientProvider client={queryClient}>
          <ThemeProvider defaultTheme="system" storageKey="trenova-dash-theme">
            <DriverAppPlanGate>
              <RouterProvider router={dashRouter} />
            </DriverAppPlanGate>
            <Toaster position="top-center" />
          </ThemeProvider>
        </QueryClientProvider>
      </RootErrorBoundary>
    </LazyMotion>
  </StrictMode>,
);
