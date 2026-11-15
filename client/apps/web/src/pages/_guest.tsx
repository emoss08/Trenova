import { createFileRoute } from "@tanstack/react-router";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { SHOWN_AT_ONCE, guestGate } from "@/lib/route-gates";
import { gates } from "@trenova/shared/lib/router/gates";

export const Route = createFileRoute("/_guest")({
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: gates(guestGate),
});
