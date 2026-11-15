import { createFileRoute } from "@tanstack/react-router";
import { onboardingGate } from "@/lib/route-gates";
import { OnboardingPage } from "@/routes/onboarding/page";
import { gates } from "@trenova/shared/lib/router/gates";

export const Route = createFileRoute("/_onboarding/onboarding")({
  beforeLoad: gates(onboardingGate),
  component: OnboardingPage,
});
