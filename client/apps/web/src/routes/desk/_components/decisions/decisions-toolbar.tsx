import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { humanizeToolName } from "@/components/assistant/proposal-state";
import { XIcon } from "lucide-react";
import { useMemo } from "react";
import { usePendingDecisionSummary, type PendingDecisionFilter } from "./use-pending-decisions";

const ALL = "__all__";

/**
 * The queue's filters, read from the summary so they only ever name agents
 * and tools that have something waiting. The count on the right is the
 * queue as filtered.
 */
export function DecisionsToolbar({
  filter,
  totalCount,
  onFilterChange,
}: {
  filter: PendingDecisionFilter;
  totalCount: number | null;
  onFilterChange: (next: PendingDecisionFilter) => void;
}) {
  const t = useT();
  const summary = usePendingDecisionSummary();

  const agentItems = useMemo(
    () => [
      { value: ALL, label: t("All agents") },
      ...(summary.data?.byAgent ?? []).map((row) => ({
        value: row.agentDefinitionId,
        label: row.agentName || t("Retired agent"),
        caption: String(row.count),
      })),
    ],
    [summary.data?.byAgent, t],
  );
  const toolItems = useMemo(
    () => [
      { value: ALL, label: t("All changes") },
      ...(summary.data?.byTool ?? []).map((row) => ({
        value: row.toolName,
        label: row.toolName === "plan" ? t("Plans") : humanizeToolName(row.toolName),
        caption: String(row.count),
      })),
    ],
    [summary.data?.byTool, t],
  );

  const filtered = Boolean(filter.agentDefinitionId || filter.toolName);

  return (
    <div className="border-border flex min-h-11 flex-wrap items-center gap-2 border-b px-3 py-1.5">
      {agentItems.length > 1 && (
        <SegmentedControl
          items={agentItems}
          value={filter.agentDefinitionId || ALL}
          onValueChange={(value) =>
            onFilterChange({ ...filter, agentDefinitionId: value === ALL ? "" : value })
          }
        />
      )}
      {toolItems.length > 1 && (
        <SegmentedControl
          items={toolItems}
          value={filter.toolName || ALL}
          onValueChange={(value) => onFilterChange({ ...filter, toolName: value === ALL ? "" : value })}
        />
      )}
      {filtered && (
        <Button
          size="xs"
          variant="ghost"
          className="text-muted-foreground"
          onClick={() => onFilterChange({ agentDefinitionId: "", toolName: "" })}
        >
          <XIcon className="size-3" />
          {t("Clear")}
        </Button>
      )}
      <span className="text-muted-foreground ml-auto text-xs tabular-nums">
        {totalCount === null
          ? ""
          : t("{0, plural, one {# waiting} other {# waiting}}", totalCount)}
      </span>
    </div>
  );
}
