import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import type { WatchtowerAgentStep } from "@/lib/graphql/watchtower";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

export function stepName(t: TranslateFn, step: WatchtowerAgentStep): string {
  switch (step) {
    case "Read":
      return t("Read");
    case "Decide":
      return t("Decide");
    default:
      return t("Approve");
  }
}

/** A run's real progress: steps done, the one under way, and what is left. */
export function RunSteps({ work }: { work: NonNullable<DeskAttentionItem["agentWork"]> }) {
  const t = useT();
  const current = Math.min(work.step + 1, work.steps.length);

  return (
    <span className="dk-w-steps">
      <span
        className="dk-w-steps-b"
        style={{ gridTemplateColumns: `repeat(${work.steps.length}, 1fr)` }}
        aria-label={t("Step {0} of {1}", current, work.steps.length)}
      >
        {work.steps.map((step, index) => (
          <i
            key={step}
            className={cn(
              index < work.step && "dk-on",
              index === work.step && work.status === "Working" && "dk-cur",
            )}
          />
        ))}
      </span>
      <span className="dk-w-steps-l">
        {work.steps.map((step) => (
          <span key={step}>{stepName(t, step)}</span>
        ))}
      </span>
    </span>
  );
}
