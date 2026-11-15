import { createFileRoute } from "@tanstack/react-router";
import LoadingSkeleton from "@trenova/shared/components/loading-skeleton";
import { SHOWN_AT_ONCE, protectedGate } from "@/lib/route-gates";
import { OnboardingLayout } from "@/routes/app-layout";

// The welcome wizard a new cloud organization walks through once. It is a
// full-screen step of its own rather than a page in the app: nothing in the
// sidebar is useful until the organization has a profile. It still runs the
// session's gates, so a password change or role activation comes first.
export const Route = createFileRoute("/_onboarding")({
  pendingComponent: LoadingSkeleton,
  ...SHOWN_AT_ONCE,
  beforeLoad: protectedGate(),
  component: OnboardingLayout,
});
