import { queries } from "@/lib/queries";
import { agentControlQueryOptions } from "@/lib/graphql/agent-control";
import type { AutonomyTier, ToolCatalogEntry, ToolTrust } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Progress } from "@trenova/shared/components/ui/progress";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { AwardIcon } from "lucide-react";
import { useMemo } from "react";
import { tierWithin } from "./agent-form-schema";
import { TIER_LABEL, TIER_ORDER } from "./tool-catalog";
import { describeToolCall } from "@/components/assistant/tool-presentation";

export type TrustSettings = {
  earnedAutonomy: boolean;
  threshold: number;
  ceiling: AutonomyTier;
};

export type TrustSummary =
  | { state: "counting"; remaining: number; nextTier: AutonomyTier }
  | { state: "earned" }
  | { state: "at-ceiling" }
  | { state: "off" };

function isTier(value: string): value is AutonomyTier {
  return (TIER_ORDER as readonly string[]).includes(value);
}

/**
 * The tier the server runs a tool at: the agent's own override, else the
 * tool's default from the catalog, else Propose, and never above the
 * ceiling. This is the tier the ledger promotes from, so it is computed the
 * way the runtime computes it rather than the way the picker shows it.
 */
export function runtimeTier(
  toolName: string,
  tool: ToolCatalogEntry | undefined,
  tiers: Record<string, AutonomyTier>,
  ceiling: AutonomyTier,
): AutonomyTier {
  const fallback = tool && isTier(tool.defaultAutonomyTier) ? tool.defaultAutonomyTier : "Propose";
  const own = tiers[toolName] ?? fallback;
  return tierWithin(own, ceiling) ? own : ceiling;
}

function nextTier(current: AutonomyTier): AutonomyTier | null {
  const index = TIER_ORDER.indexOf(current);
  return index >= 0 && index < TIER_ORDER.length - 1 ? TIER_ORDER[index + 1] : null;
}

/**
 * What stands between a tool and its next tier. The order matters: a tier
 * the ledger granted is reported as earned whatever else is true, a tool
 * with nowhere to go is at the ceiling, and only then does the switch or the
 * count come into it.
 */
export function describeTrust(
  row: ToolTrust,
  current: AutonomyTier,
  settings: TrustSettings,
): TrustSummary {
  if (row.earnedTier && row.earnedTier === current) {
    return { state: "earned" };
  }

  const next = nextTier(current);
  if (next === null || TIER_ORDER.indexOf(next) > TIER_ORDER.indexOf(settings.ceiling)) {
    return { state: "at-ceiling" };
  }

  if (!settings.earnedAutonomy) {
    return { state: "off" };
  }

  return {
    state: "counting",
    remaining: Math.max(settings.threshold - row.streak, 0),
    nextTier: next,
  };
}

const LAST_DECISION_FORMAT = {
  month: "short",
  day: "numeric",
  hour: "numeric",
  minute: "2-digit",
} as const;

type TrackRecordProps = {
  rows: readonly ToolTrust[];
  tools: readonly ToolCatalogEntry[];
  tiers: Record<string, AutonomyTier>;
  ceiling: AutonomyTier;
  earnedAutonomy: boolean;
  threshold: number;
};

/**
 * How the agent's change tools have been decided on: the streak against the
 * threshold, the totals, and any tier that was earned rather than chosen.
 */
export function TrackRecord({
  rows,
  tools,
  tiers,
  ceiling,
  earnedAutonomy,
  threshold,
}: TrackRecordProps) {
  const t = useT();
  const catalog = useMemo(() => new Map(tools.map((tool) => [tool.name, tool])), [tools]);

  if (rows.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("No decisions yet. The ledger starts with the first proposal someone decides on.")}
      </p>
    );
  }

  return (
    <ul className="divide-border flex flex-col divide-y">
      {rows.map((row) => {
        const tool = catalog.get(row.toolName);
        const title = describeToolCall(row.toolName, null).title;
        const current = runtimeTier(row.toolName, tool, tiers, ceiling);
        const summary = describeTrust(row, current, { earnedAutonomy, threshold, ceiling });

        return (
          <li key={row.id} className="flex flex-col gap-1.5 py-2.5 first:pt-0 last:pb-0">
            <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
              <span className="text-sm font-medium">{title}</span>
              {tool ? null : (
                <span className="text-muted-foreground text-xs">{t("No longer offered")}</span>
              )}
              <Badge variant="neutral" className="h-4 px-1.5 text-2xs">
                {t(TIER_LABEL[current])}
              </Badge>
              {summary.state === "earned" && (
                <Badge variant="success" className="h-4 gap-1 px-1.5 text-2xs">
                  <AwardIcon className="size-2.5" />
                  {t("Earned")}
                </Badge>
              )}
            </div>
            {summary.state === "counting" && (
              <div className="flex items-center gap-3">
                <Progress
                  value={row.streak}
                  max={threshold}
                  size="sm"
                  className="max-w-40"
                  aria-label={t("Clean approvals for {0}", title)}
                />
                <span className="text-muted-foreground text-xs tabular-nums">
                  {t(
                    "{0} more clean approvals to {1}",
                    summary.remaining,
                    t(TIER_LABEL[summary.nextTier]),
                  )}
                </span>
              </div>
            )}
            {summary.state === "at-ceiling" && (
              <span className="text-muted-foreground text-xs">
                {t("At the agent's ceiling; raise it to let this tool earn more.")}
              </span>
            )}
            {summary.state === "off" && (
              <span className="text-muted-foreground text-xs">
                {t(
                  "Earned autonomy is off for the organization; the streak is counted but changes nothing.",
                )}
              </span>
            )}
            <span className="text-muted-foreground text-xs tabular-nums">
              {t(
                "{0} approved · {1} changed · {2} rejected · {3} failed",
                row.approvals,
                row.modifications,
                row.rejections,
                row.executionFailures,
              )}
              {row.lastDecisionAt ? (
                <>
                  {" · "}
                  {t(
                    "last {0}",
                    formatUnixInUserTimezone(row.lastDecisionAt, LAST_DECISION_FORMAT, ""),
                  )}
                </>
              ) : null}
            </span>
          </li>
        );
      })}
    </ul>
  );
}

type TrackRecordSectionProps = {
  agentId: string;
  tools: readonly ToolCatalogEntry[];
  tiers: Record<string, AutonomyTier>;
  ceiling: AutonomyTier;
};

/** The ledger for one saved agent, with the organization's switch and threshold. */
export function TrackRecordSection({ agentId, tools, tiers, ceiling }: TrackRecordSectionProps) {
  const trustQuery = useQuery(queries.assistant.agentTrust(agentId));
  const controlQuery = useQuery(agentControlQueryOptions());

  if (trustQuery.isLoading || controlQuery.isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-4 w-1/3" />
      </div>
    );
  }

  return (
    <TrackRecord
      rows={trustQuery.data?.results ?? []}
      tools={tools}
      tiers={tiers}
      ceiling={ceiling}
      earnedAutonomy={controlQuery.data?.earnedAutonomy ?? false}
      threshold={controlQuery.data?.promotionThreshold ?? 10}
    />
  );
}
