import { integrationSetupStepStates, type IntegrationSetupStep } from "@/lib/integration-setup";
import { Stepper } from "@trenova/shared/components/ui/stepper";
import type { ReactNode } from "react";

type IntegrationSetupWizardProps = {
  steps: readonly IntegrationSetupStep[];
  activeStepId: string;
  label: string;
  children: ReactNode;
};

export function IntegrationSetupWizard({
  steps,
  activeStepId,
  label,
  children,
}: IntegrationSetupWizardProps) {
  return (
    <div className="grid gap-6 sm:grid-cols-[11rem_minmax(0,1fr)]">
      <Stepper steps={integrationSetupStepStates(steps, activeStepId)} aria-label={label} />
      <section className="min-w-0 space-y-4">{children}</section>
    </div>
  );
}
