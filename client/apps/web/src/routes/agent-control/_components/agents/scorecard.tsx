import {
  SCORECARD_WINDOWS,
  type AgentScorecard,
  type AgentScorecardWindow,
  type AgentToolOutcome,
} from "@/lib/graphql/agent-scorecard";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { describeToolCall } from "@/components/assistant/tool-presentation";

/**
 * Whether an agent is worth keeping, in one panel.
 *
 * The figures answer that question in the order a person asks it: how much
 * did it do, how much of it was right, how much did it cost, and what did it
 * take off somebody. Every one is counted at read time from the runs, the
 * proposals and the usage records, so nothing here can disagree with the
 * lists those come from.
 */
export function AgentScorecardPanel({ agentDefinitionId }: { agentDefinitionId: string }) {
  const t = useT();
  const [window, setWindow] = useState<AgentScorecardWindow>("Last30Days");
  const { data, isLoading } = useQuery(queries.agentScorecard.detail(agentDefinitionId, window));

  return (
    <section className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <h3 className="text-muted-foreground text-xs font-medium">{t("Track record")}</h3>
        <div className="ml-auto flex items-center gap-1">
          {SCORECARD_WINDOWS.map((option) => (
            <button
              key={option}
              type="button"
              aria-pressed={option === window}
              onClick={() => setWindow(option)}
              className={cn(
                "ui-focus-ring rounded-full px-2 py-0.5 text-xs transition-colors",
                option === window
                  ? "bg-foreground text-background"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {windowLabel(t, option)}
            </button>
          ))}
        </div>
      </div>

      {isLoading || data === undefined ? (
        <Skeleton className="h-28" />
      ) : (
        <ScorecardBody scorecard={data} />
      )}
    </section>
  );
}

function ScorecardBody({ scorecard }: { scorecard: AgentScorecard }) {
  const t = useT();

  if (scorecard.runs === 0 && scorecard.proposals === 0) {
    return (
      <p className="text-muted-foreground border-border rounded-surface border px-4 py-6 text-center text-sm">
        {t("This agent has not run in this window.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <dl className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4">
        <Figure label={t("Runs")} value={String(scorecard.runs)} />
        <Figure
          label={t("Approved")}
          value={
            scorecard.approvalRate === null || scorecard.approvalRate === undefined
              ? t("Not yet")
              : `${Math.round(scorecard.approvalRate * 100)}%`
          }
          // Pending is not a rejection. An agent whose proposals are all
          // still waiting has an unknown record, not a bad one.
          hint={
            scorecard.pending > 0
              ? t("{0, plural, one {# waiting} other {# waiting}}", scorecard.pending)
              : undefined
          }
        />
        <Figure label={t("Cost")} value={`$${scorecard.costUsd}`} />
        <Figure
          label={t("Time saved")}
          value={formatMinutes(t, scorecard.estimatedMinutesSaved)}
          hint={t("estimate")}
        />
      </dl>

      {(scorecard.runsFailed > 0 ||
        scorecard.exceptions > 0 ||
        scorecard.executionFailures > 0) && (
        <div className="flex flex-wrap gap-1.5">
          {scorecard.runsFailed > 0 && (
            <Badge variant="danger">
              {t("{0, plural, one {# failed run} other {# failed runs}}", scorecard.runsFailed)}
            </Badge>
          )}
          {scorecard.executionFailures > 0 && (
            <Badge variant="danger">
              {t(
                "{0, plural, one {# approved change did not run} other {# approved changes did not run}}",
                scorecard.executionFailures,
              )}
            </Badge>
          )}
          {scorecard.exceptions > 0 && (
            <Badge variant="warning">
              {t(
                "{0, plural, one {# exception raised} other {# exceptions raised}}",
                scorecard.exceptions,
              )}
            </Badge>
          )}
        </div>
      )}

      {scorecard.autoExecuted > 0 && (
        <p className="text-muted-foreground text-xs">
          {t(
            "{0, plural, one {# change ran without being asked} other {# changes ran without being asked}}",
            scorecard.autoExecuted,
          )}
        </p>
      )}

      {scorecard.byTool.length > 0 && <ToolBreakdown tools={scorecard.byTool} />}
    </div>
  );
}

/**
 * Per tool, because one total hides a bad tool inside nine good ones — and
 * "which of this agent's tools is being rejected" is the question somebody
 * deciding whether to narrow its reach actually has.
 */
function ToolBreakdown({ tools }: { tools: readonly AgentToolOutcome[] }) {
  const t = useT();

  return (
    <div className="border-border rounded-surface divide-border divide-y border">
      {tools.map((tool) => {
        const decided = tool.approved + tool.modified + tool.rejected;

        return (
          <div key={tool.toolName} className="flex items-center gap-3 px-3 py-2">
            <span className="min-w-0 flex-1 truncate text-sm">
              {describeToolCall(tool.toolName, null).title}
            </span>
            <span className="text-muted-foreground flex shrink-0 items-center gap-2 text-xs tabular-nums">
              {decided > 0 && <span>{t("{0} of {1} approved", tool.approved, decided)}</span>}
              {tool.rejected > 0 && (
                <Badge variant="neutral" className="h-4 px-1">
                  {t("{0} rejected", tool.rejected)}
                </Badge>
              )}
              {tool.failed > 0 && (
                <Badge variant="danger" className="h-4 px-1">
                  {t("{0} failed", tool.failed)}
                </Badge>
              )}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function Figure({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className="flex items-baseline gap-1.5">
        <span className="text-lg tabular-nums">{value}</span>
        {hint && <span className="text-muted-foreground text-xs">{hint}</span>}
      </dd>
    </div>
  );
}

function formatMinutes(t: (value: string, ...args: unknown[]) => string, minutes: number): string {
  if (minutes < 60) {
    return t("{0}m", minutes);
  }
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;

  return rest === 0 ? t("{0}h", hours) : t("{0}h {1}m", hours, rest);
}

function windowLabel(
  t: (value: string, ...args: unknown[]) => string,
  window: AgentScorecardWindow,
): string {
  switch (window) {
    case "Last7Days":
      return t("7 days");
    case "Last90Days":
      return t("90 days");
    default:
      return t("30 days");
  }
}
