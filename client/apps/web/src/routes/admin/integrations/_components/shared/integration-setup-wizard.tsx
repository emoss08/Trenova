import { integrationSetupStepStates, type IntegrationSetupStep } from "@/lib/integration-setup";
import { Stepper } from "@trenova/shared/components/ui/stepper";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

type IntegrationSetupWizardProps = {
  steps: readonly IntegrationSetupStep[];
  activeStepId: string;
  label: string;
  children: ReactNode;
};

/**
 * A setup flow that fills the rest of its dialog: the steps in a tinted rail on the left and
 * the active step beside them, scrolling on its own. A phone has no room for the rail, so it
 * shows the step alone; each step's heading says which it is. A step ends with an
 * `IntegrationSetupFooter`, which stays pinned to the bottom however long the step runs.
 */
export function IntegrationSetupWizard({
  steps,
  activeStepId,
  label,
  children,
}: IntegrationSetupWizardProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col sm:grid sm:grid-cols-[13rem_minmax(0,1fr)]">
      <aside className="bg-muted/40 hidden overflow-y-auto border-r px-5 py-5 sm:block">
        <Stepper steps={integrationSetupStepStates(steps, activeStepId)} aria-label={label} />
      </aside>
      {/* A pinned footer meets the bottom edge; a step without one keeps the bottom gutter. */}
      <section className="flex min-h-0 min-w-0 flex-1 flex-col gap-4 overflow-y-auto px-5 pt-5 pb-5 has-[[data-slot=integration-setup-footer]]:pb-0">
        {children}
      </section>
    </div>
  );
}

/**
 * The actions that end a setup step: pinned to the bottom of the step and spanning its full
 * width, so the way forward stays in view on a step that scrolls.
 */
export function IntegrationSetupFooter({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      data-slot="integration-setup-footer"
      className={cn(
        "bg-card sticky bottom-0 z-10 -mx-5 mt-auto flex flex-wrap items-center justify-end gap-2 border-t px-5 py-3",
        className,
      )}
    >
      {children}
    </div>
  );
}
