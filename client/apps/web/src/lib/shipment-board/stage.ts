import type { ShipmentStage } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BadgeTone } from "@trenova/shared/types/badge";
import type { DataTableGroup, DataTableGroupKey } from "@trenova/shared/types/data-table";
import { formatCurrency } from "@trenova/shared/lib/utils";
import type { ShipmentStageSummary } from "@/lib/graphql/shipment-board";

/** The API field the board groups, sorts and filters stages by. */
export const STAGE_GROUP_FIELD = "stageRank";

type StageMeta = {
  label: string;
  tone: BadgeTone;
  swatchClassName: string;
  textClassName: string;
};

/* The stages read as a severity ordering, from what needs a hand now to what
   is finished, so each takes a tone rather than a categorical accent. */
export const STAGE_META: Record<ShipmentStage, StageMeta> = {
  Late: {
    label: "Needs attention",
    tone: "danger",
    swatchClassName: "bg-danger",
    textClassName: "text-danger",
  },
  NeedsCoverage: {
    label: "Needs coverage",
    tone: "warning",
    swatchClassName: "bg-warning",
    textClassName: "text-warning",
  },
  Moving: {
    label: "Moving",
    tone: "info",
    swatchClassName: "bg-info",
    textClassName: "text-info",
  },
  Scheduled: {
    label: "Scheduled",
    tone: "neutral",
    swatchClassName: "bg-muted-foreground",
    textClassName: "text-muted-foreground",
  },
  Delivered: {
    label: "Delivered",
    tone: "success",
    swatchClassName: "bg-success",
    textClassName: "text-success",
  },
  Canceled: {
    label: "Canceled",
    tone: "neutral",
    swatchClassName: "bg-border-strong",
    textClassName: "text-muted-foreground",
  },
};

function wholeDollars(value: string): string {
  return formatCurrency(Math.round(Number(value) || 0)).replace(/\.00$/, "");
}

/** Group headers for the stages the current filters reach, with their whole-group totals. */
export function stageGroups(summary: readonly ShipmentStageSummary[], t: TranslateFn): DataTableGroup[] {
  return summary
    .filter((entry) => entry.count > 0)
    .sort((a, b) => a.rank - b.rank)
    .map((entry) => ({
      key: entry.rank,
      label: t(STAGE_META[entry.stage].label),
      swatchClassName: STAGE_META[entry.stage].swatchClassName,
      count: entry.count,
      aggregate: wholeDollars(entry.revenue),
    }));
}

/** A row's group key: the server's rank for its stage, so client and server never disagree on order. */
export function stageRankLookup(summary: readonly ShipmentStageSummary[]) {
  const ranks = new Map<ShipmentStage, number>(summary.map((entry) => [entry.stage, entry.rank]));
  return (stage: ShipmentStage | undefined | null): DataTableGroupKey =>
    stage ? (ranks.get(stage) ?? stage) : "unknown";
}
