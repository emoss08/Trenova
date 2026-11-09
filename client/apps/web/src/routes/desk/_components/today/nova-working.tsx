import { DeskAgentTile } from "@/components/desk-chat/desk-agent-tile";
import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import type { WatchtowerAgentStep } from "@/lib/graphql/watchtower";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";

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

/** The run's real progress: steps done, the one under way, and what is left. */
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

/**
 * The work agents have in hand right now, each with its run's real progress:
 * reading the record, working out what to do, asking a person to approve. The
 * bar moves only when the run does; nothing here is simulated.
 */
export function NovaWorking({
  items,
  pathOf,
}: {
  items: readonly DeskAttentionItem[];
  /** Where a run opens: the case of the record it works on. */
  pathOf: (item: DeskAttentionItem) => string;
}) {
  const t = useT();

  return (
    <div className="dk-w-rows">
      {items.map((item) => {
        const work = item.agentWork;
        if (!work) {
          return null;
        }
        return (
          <Link key={item.id} to={pathOf(item)} className="dk-w-run ui-focus-ring">
            <DeskAgentTile agent={work.agent} size="sm" />
            <span className="dk-w-run-main">
              <span className="dk-w-run-ti">{item.title}</span>
              <span className="dk-w-run-nt">
                {work.note !== "" ? work.note : (work.agent?.name ?? t("An agent"))}
              </span>
            </span>
            <RunSteps work={work} />
          </Link>
        );
      })}
    </div>
  );
}
