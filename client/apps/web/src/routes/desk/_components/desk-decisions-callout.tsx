import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ArrowRightIcon, InboxIcon } from "lucide-react";
import { Link } from "react-router";

/** Agents named before the rest are folded into "and N more". */
const NAMED_AGENTS = 3;

export type DecisionAgentCount = {
  agentDefinitionId: string;
  agentName: string;
  count: number;
};

export type DeskDecisionsCalloutProps = {
  waiting: number;
  byAgent: readonly DecisionAgentCount[];
  /** When the oldest waiting decision arrived, in Unix seconds. */
  oldestAt: number | null;
  agentsById: ReadonlyMap<string, AgentChoice>;
  now: number;
  className?: string;
};

/**
 * The one thing on the front page that is somebody else waiting on you.
 *
 * It says how many, who is asking and how long the oldest has waited, and
 * then it offers the one action there is. The count is the same number the
 * headline, the launcher and the sidebar show; it is never a second opinion.
 */
export function DeskDecisionsCallout({
  waiting,
  byAgent,
  oldestAt,
  agentsById,
  now,
  className,
}: DeskDecisionsCalloutProps) {
  const t = useT();
  const named = byAgent.slice(0, NAMED_AGENTS);
  const others = byAgent.length - named.length;

  return (
    <section
      aria-labelledby="desk-decisions-heading"
      className={cn(
        "border-desk-hairline rounded-surface flex flex-col gap-3 border p-4",
        className,
      )}
    >
      <div className="flex items-start gap-3">
        <span className="bg-sunken text-muted-foreground flex size-9 shrink-0 items-center justify-center rounded-md">
          <InboxIcon className="size-4" />
        </span>
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="text-2xl font-semibold tabular-nums">{waiting}</span>
          <h2 id="desk-decisions-heading" className="text-sm font-semibold">
            {t("Waiting on your decision")}
          </h2>
          {oldestAt !== null && oldestAt > 0 && (
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("Oldest {0}", formatSecondsAgo(now - oldestAt))}
            </span>
          )}
        </div>
      </div>

      {named.length > 0 && (
        <ul className="flex flex-col gap-1">
          {named.map((row) => (
            <li key={row.agentDefinitionId} className="flex items-center gap-2 text-sm">
              <AgentTile
                agent={
                  agentsById.get(row.agentDefinitionId) ?? {
                    id: row.agentDefinitionId,
                    name: row.agentName,
                  }
                }
                size="xs"
              />
              <span className="min-w-0 flex-1 truncate">{row.agentName || t("Retired agent")}</span>
              <span className="text-muted-foreground tabular-nums">{row.count}</span>
            </li>
          ))}
          {others > 0 && (
            <li className="text-muted-foreground pl-7 text-xs">
              {t("{0, plural, one {and one more agent} other {and # more agents}}", others)}
            </li>
          )}
        </ul>
      )}

      <Button nativeButton={false} render={<Link to="/desk/decisions" />} className="group w-full">
        {t("Review decisions")}
        <ArrowRightIcon className="transition-transform group-hover:translate-x-0.5" />
      </Button>
    </section>
  );
}
