import { queries } from "@/lib/queries";
import { formatUsd } from "@/lib/ai-usage-format";
import { NumberField } from "@/components/fields/number-field";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import type { AgentBudgetStatus, ToolCatalogEntry } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";
import { useFormContext } from "react-hook-form";
import type { AgentFormValues } from "./agent-form-schema";

/**
 * A daily cap for each change tool the agent holds. Reads are not listed:
 * a cap on a lookup would only make the agent answer worse, and the cost
 * of lookups is already under the monthly budget. A tool the catalog does
 * not describe is not listed either, since nothing says it changes anything.
 */
export function ToolLimitsField({
  toolNames,
  tools,
}: {
  toolNames: readonly string[];
  tools: readonly ToolCatalogEntry[];
}) {
  const t = useT();
  const { control } = useFormContext<AgentFormValues>();
  const writes = useMemo(() => {
    const catalog = new Map(tools.map((tool) => [tool.name, tool]));
    return toolNames.filter((name) => catalog.get(name)?.kind === "action");
  }, [toolNames, tools]);

  if (writes.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("Give the agent a change tool to set a daily limit on it.")}
      </p>
    );
  }

  return (
    <ul className="divide-border flex flex-col divide-y">
      {writes.map((name) => (
        <li key={name} className="flex items-center gap-3 py-2 first:pt-0 last:pb-0">
          <span className="min-w-0 flex-1 truncate text-sm">
            {describeToolCall(name, null).title}
          </span>
          <div className="w-36 shrink-0">
            <NumberField
              name={`toolDailyLimits.${name}`}
              control={control}
              aria-label={t("Daily limit for {0}", describeToolCall(name, null).title)}
              placeholder={t("No limit")}
              min={0}
              max={10000}
              sideText={t("per day")}
            />
          </div>
        </li>
      ))}
    </ul>
  );
}

/** How much of a cap is spent, for a progress bar: 0 when there is no cap. */
export function budgetShare(spent: number, limit: number | null | undefined): number {
  if (!limit || limit <= 0) return 0;
  return Math.min(100, Math.round((spent / limit) * 100));
}

export function BudgetStatusSection({ agentId }: { agentId: string }) {
  const t = useT();
  const query = useQuery(queries.assistant.agentBudget(agentId));

  if (query.isLoading) {
    return <Skeleton className="h-16 w-full" />;
  }
  if (!query.data) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("Where the agent stands against its caps could not be read.")}
      </p>
    );
  }

  return <BudgetStatus status={query.data} />;
}

export function BudgetStatus({ status }: { status: AgentBudgetStatus }) {
  const t = useT();
  const spent = Number(status.spentUsd);
  const budget = status.monthlyBudgetUsd ? Number(status.monthlyBudgetUsd) : null;

  return (
    <ul className="flex flex-col gap-3">
      <li className="flex flex-col gap-1">
        <div className="flex items-baseline justify-between gap-3 text-sm">
          <span>{t("This month")}</span>
          <span className="text-muted-foreground text-xs tabular-nums">
            {budget === null
              ? t("{0} spent, no cap", formatUsd(status.spentUsd) ?? "$0.00")
              : t(
                  "{0} of {1}",
                  formatUsd(status.spentUsd) ?? "$0.00",
                  formatUsd(status.monthlyBudgetUsd) ?? "—",
                )}
            {status.unpricedCalls > 0 ? ` · ${t("{0} calls unpriced", status.unpricedCalls)}` : ""}
          </span>
        </div>
        {budget !== null && (
          <Progress
            value={budgetShare(spent, budget)}
            size="sm"
            aria-label={t("Monthly budget spent")}
          />
        )}
      </li>
      <li className="flex flex-col gap-1">
        <div className="flex items-baseline justify-between gap-3 text-sm">
          <span>{t("Runs today")}</span>
          <span className="text-muted-foreground text-xs tabular-nums">
            {status.dailyRunLimit > 0
              ? t("{0} of {1}", status.runsToday, status.dailyRunLimit)
              : t("{0}, no cap", status.runsToday)}
          </span>
        </div>
        {status.dailyRunLimit > 0 && (
          <Progress
            value={budgetShare(status.runsToday, status.dailyRunLimit)}
            size="sm"
            aria-label={t("Runs today")}
          />
        )}
      </li>
      {status.tools.map((tool) => (
        <li key={tool.tool} className="flex flex-col gap-1">
          <div className="flex items-baseline justify-between gap-3 text-sm">
            <span>{describeToolCall(tool.tool, null).title}</span>
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} of {1} today", tool.used, tool.limit)}
            </span>
          </div>
          <Progress
            value={budgetShare(tool.used, tool.limit)}
            size="sm"
            aria-label={describeToolCall(tool.tool, null).title}
          />
        </li>
      ))}
    </ul>
  );
}
