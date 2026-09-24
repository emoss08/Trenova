import type { StepperStep } from "@trenova/shared/components/ui/stepper";

export type IntegrationSetupStep = {
  id: string;
  label: string;
  detail?: string;
};

export function integrationSetupStepStates(
  steps: readonly IntegrationSetupStep[],
  activeStepId: string,
): StepperStep[] {
  const found = steps.findIndex((step) => step.id === activeStepId);
  const activeIndex = found === -1 ? 0 : found;

  return steps.map((step, index) => ({
    ...step,
    state: index < activeIndex ? "done" : index === activeIndex ? "active" : "pending",
  }));
}
