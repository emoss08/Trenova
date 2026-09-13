import { pinnedCellClass, pinnedCellStyle } from "@/lib/data-table";
import { cn } from "@trenova/shared/lib/utils";
import type { Column } from "@trenova/shared/types/data-table";
import type { ColumnPinningState, RowData } from "@tanstack/react-table";
import type { CSSProperties } from "react";

export const COMMAND_CENTER_COLUMN_PINNING: ColumnPinningState = {
  start: ["lane"],
  end: ["actions"],
};

export type CommandCenterRowTone = "default" | "highlighted" | "expanded";

const PINNED_ROW_TONE_CLASS: Record<CommandCenterRowTone, string> = {
  default: "bg-card group-hover/row:bg-[color-mix(in_oklab,var(--muted)_30%,var(--card))]",
  highlighted:
    "bg-[color-mix(in_oklab,var(--muted)_50%,var(--card))] group-hover/row:bg-[color-mix(in_oklab,var(--muted)_30%,var(--card))]",
  expanded:
    "bg-[color-mix(in_oklab,var(--brand)_10%,var(--card))] group-hover/row:bg-[color-mix(in_oklab,var(--brand)_20%,var(--card))]",
};

export function pinnedHeaderCellClass<TData extends RowData>(
  column: Column<TData>,
): string | undefined {
  if (!column.getIsPinned()) return undefined;
  return cn(pinnedCellClass(column), "bg-muted");
}

export function pinnedRowCellClass<TData extends RowData>(
  column: Column<TData>,
  tone: CommandCenterRowTone,
): string | undefined {
  if (!column.getIsPinned()) return undefined;
  return cn(pinnedCellClass(column), "transition-colors", PINNED_ROW_TONE_CLASS[tone]);
}

export function pinnedRowCellStyle<TData extends RowData>(
  column: Column<TData>,
  tone: CommandCenterRowTone,
): CSSProperties | undefined {
  const offset = pinnedCellStyle(column);
  if (!offset || tone !== "expanded") return offset;

  const pinned = column.getIsPinned();
  const shadows = ["inset 0 1px 0 0 var(--brand)", "inset 0 -1px 0 0 var(--brand)"];
  if (pinned === "start" && column.getIsFirstColumn("start")) {
    shadows.push("inset 1px 0 0 0 var(--brand)");
  }
  if (pinned === "end" && column.getIsLastColumn("end")) {
    shadows.push("inset -1px 0 0 0 var(--brand)");
  }
  if (pinned === "start" && column.getIsLastColumn("start")) {
    shadows.push("inset -1px 0 0 0 var(--border)");
  }
  if (pinned === "end" && column.getIsFirstColumn("end")) {
    shadows.push("inset 1px 0 0 0 var(--border)");
  }
  return { ...offset, boxShadow: shadows.join(", ") };
}
