import { createFileRoute } from "@tanstack/react-router";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { SHOWN_AT_ONCE, protectedGate } from "@/lib/route-gates";
import { AppErrorLayout, AppLayout } from "@/routes/app-layout";

// A page that fails to load or render keeps the shell standing, so the way to
// anywhere else is still on screen.
export const Route = createFileRoute("/_app")({
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: protectedGate(),
  component: AppLayout,
  errorComponent: AppErrorLayout,
});
