import { useT } from "@trenova/shared/i18n/use-t";
import type { DQFNextStep } from "@trenova/shared/lib/dqf";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, ChevronRightIcon } from "lucide-react";

type DQFNextStepsProps = {
  steps: readonly DQFNextStep[];
  busyId?: string | null;
  onStep: (step: DQFNextStep) => void;
};

const ACTION_HINTS: Record<string, string> = {
  request: "Send request",
  chase: "Chase again",
  close: "Close as no response",
  drugAlcohol: "Record response",
  addEmployer: "Add employer",
  review: "Retention",
};

/**
 * The work on the file, in order. Each row is the thing to do next and takes
 * the reader to where it is done — or does it on the spot when it is a
 * single act like sending a request.
 */
export function DQFNextSteps({ steps, busyId, onStep }: DQFNextStepsProps) {
  const t = useT();

  return (
    <section className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between">
        <h4 className="text-muted-foreground text-[11px] font-semibold uppercase">
          {t("Next steps")}
        </h4>
        {steps.length > 0 ? (
          <span className="text-muted-foreground font-mono text-[11px] tabular-nums">
            {steps.length}
          </span>
        ) : null}
      </div>
      {steps.length === 0 ? (
        <div className="text-muted-foreground flex items-center gap-2 rounded-lg border border-dashed px-4 py-4 text-xs">
          <CheckIcon className="size-4" />
          <span>{t("Nothing left to do — the file is complete.")}</span>
        </div>
      ) : (
        <ol className="divide-border divide-y rounded-lg border">
          {steps.map((step, index) => {
            const interactive = step.kind !== "purge";
            const hint = step.action ? ACTION_HINTS[step.action] : step.tab ? "Open" : undefined;
            const body = (
              <>
                <span
                  className={cn(
                    "inline-flex size-6 shrink-0 items-center justify-center rounded-md text-[10px] font-medium tabular-nums",
                    step.blocking ? "bg-primary text-primary-foreground" : "bg-accent",
                  )}
                  aria-hidden
                >
                  {index + 1}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium">{t(step.label)}</span>
                  <span className="text-muted-foreground block truncate text-xs">
                    {step.detail}
                  </span>
                </span>
                {hint ? (
                  <span className="text-muted-foreground hidden shrink-0 text-xs sm:inline">
                    {hint}
                  </span>
                ) : null}
                {interactive ? (
                  <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
                ) : null}
              </>
            );
            return (
              <li key={step.id}>
                {interactive ? (
                  <button
                    type="button"
                    data-testid={`dqf-step-${step.id}`}
                    disabled={busyId === step.id}
                    onClick={() => onStep(step)}
                    className="hover:bg-muted/40 flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors disabled:opacity-60"
                  >
                    {body}
                  </button>
                ) : (
                  <div
                    data-testid={`dqf-step-${step.id}`}
                    className="flex items-center gap-3 px-3 py-2.5"
                  >
                    {body}
                  </div>
                )}
              </li>
            );
          })}
        </ol>
      )}
    </section>
  );
}
