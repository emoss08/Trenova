import { SectionPanel } from "@/components/section-panel";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import type { DQStep, StepAction } from "./dq-file-model";

/**
 * The work on the file as a numbered list, because it is a sequence: each step says what to do
 * and carries the one action that does it, or takes the reader to where it is done.
 */
export function DQNextSteps({
  steps,
  busyId,
  onAction,
}: {
  steps: readonly DQStep[];
  /** The step whose action is running, so its button holds. */
  busyId: string | null;
  onAction: (step: DQStep, action: StepAction) => void;
}) {
  const t = useT();
  return (
    <SectionPanel title={t("Next steps")} count={steps.length}>
      {steps.length === 0 ? (
        <p className="flex items-center gap-2 px-3 py-3 text-xs text-muted-foreground">
          <CheckIcon aria-hidden className="size-3.5 text-success-foreground" />
          {t("Nothing left to do. The file is complete.")}
        </p>
      ) : (
        <ol className="py-1">
          {steps.map((step, index) => (
            <li
              key={step.id}
              data-testid={`dqf-step-${step.id}`}
              className="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-2.5 px-3 py-2.5 not-first:border-t not-first:border-border-subtle"
            >
              <span
                aria-hidden
                className="grid size-5 place-items-center rounded-sm border font-mono text-xs text-muted-foreground"
              >
                {index + 1}
              </span>
              <div className="flex min-w-0 flex-col items-start gap-2">
                <p className="text-sm">{step.text}</p>
                {step.action ? (
                  <Button
                    type="button"
                    size="xs"
                    variant={index === 0 ? "default" : "secondary"}
                    isLoading={busyId === step.id}
                    onClick={() => step.action && onAction(step, step.action)}
                  >
                    {step.action.label}
                  </Button>
                ) : null}
              </div>
            </li>
          ))}
        </ol>
      )}
    </SectionPanel>
  );
}
