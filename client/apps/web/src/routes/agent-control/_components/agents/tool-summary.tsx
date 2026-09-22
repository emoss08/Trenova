import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { VirtualRows, type VirtualRow } from "@/components/virtual-rows";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { PencilLineIcon, SlidersHorizontalIcon, XIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import {
  TIER_LABEL,
  effectiveTier,
  groupToolsByResource,
  splitCoreTools,
  summarizeSelection,
  toggleTool,
  toolTitle,
} from "./tool-catalog";
import { ToolPickerDialog } from "./tool-picker-dialog";

type ToolSummaryProps = {
  tools: readonly ToolCatalogEntry[];
  isLoading?: boolean;
  selected: string[];
  tiers: Record<string, AutonomyTier>;
  ceiling: AutonomyTier;
  onSelectedChange: (next: string[]) => void;
  onTiersChange: (next: Record<string, AutonomyTier>) => void;
};

/**
 * What the agent may call, as the form shows it: one line of totals, then
 * the chosen tools by what they touch, each removable in place. The catalog
 * itself opens in a dialog, so the form stays the length of a form.
 */
export function ToolSummary({
  tools,
  isLoading = false,
  selected,
  tiers,
  ceiling,
  onSelectedChange,
  onTiersChange,
}: ToolSummaryProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const { core, selectable } = useMemo(() => splitCoreTools(tools), [tools]);
  const summary = useMemo(
    () => summarizeSelection(selected, tools, tiers, ceiling),
    [ceiling, selected, tiers, tools],
  );
  const chosenGroups = useMemo(() => {
    const chosen = new Set(selected);
    return groupToolsByResource(
      selectable.filter((tool) => chosen.has(tool.name)),
      selected,
    );
  }, [selectable, selected]);
  const chosenCount = summary.reads + summary.changes;

  const remove = useCallback(
    (name: string) => {
      const next = toggleTool(selected, tiers, name, false);
      onSelectedChange(next.selected);
      onTiersChange(next.tiers);
    },
    [onSelectedChange, onTiersChange, selected, tiers],
  );

  const rows = useMemo<VirtualRow[]>(() => {
    const list: VirtualRow[] = chosenGroups.map((group) => ({
      key: group.resource,
      render: () => (
        <div className="flex flex-wrap items-center gap-1.5 px-3 py-2">
          <span className="text-muted-foreground mr-1 shrink-0 text-xs font-medium">
            {group.label}
          </span>
          {group.tools.map((tool) => {
            const tier = effectiveTier(tool.name, tiers, ceiling);
            return (
              <span
                key={tool.name}
                className="border-border inline-flex h-6 max-w-full items-center gap-1 rounded-full border pl-2 text-xs"
              >
                {tool.kind === "action" && (
                  <PencilLineIcon className="text-warning-foreground size-3 shrink-0" />
                )}
                <span className="truncate">{toolTitle(tool)}</span>
                {tool.kind === "action" && (
                  <Badge variant="neutral" appearance="outline" className="h-4 px-1 text-2xs">
                    {t(TIER_LABEL[tier])}
                  </Badge>
                )}
                <button
                  type="button"
                  aria-label={t("Remove {0}", toolTitle(tool))}
                  onClick={() => remove(tool.name)}
                  className="text-muted-foreground hover:text-foreground ui-focus-ring flex h-full items-center rounded-r-full px-1.5"
                >
                  <XIcon className="size-3" />
                </button>
              </span>
            );
          })}
        </div>
      ),
    }));
    if (summary.unknown.length > 0) {
      list.push({
        key: "unknown",
        render: () => (
          <p className="text-muted-foreground px-3 py-2 text-xs">
            {t(
              "{0, plural, one {# tool the catalog no longer offers is kept on the agent and ignored.} other {# tools the catalog no longer offers are kept on the agent and ignored.}}",
              summary.unknown.length,
            )}
          </p>
        ),
      });
    }

    return list;
  }, [ceiling, chosenGroups, remove, summary.unknown.length, t, tiers]);

  if (isLoading) {
    return <Skeleton className="h-16" />;
  }

  const totals =
    chosenCount === 0
      ? t("No task tools. It can still recall, remember and escalate.")
      : [
          t("{0, plural, one {# read} other {# reads}}", summary.reads),
          summary.changes > 0
            ? t("{0, plural, one {# change} other {# changes}}", summary.changes)
            : null,
          summary.byTier.AutoExecute > 0 ? t("{0} automatic", summary.byTier.AutoExecute) : null,
        ]
          .filter((part): part is string => part !== null)
          .join(" · ");

  return (
    <div className="border-border bg-card flex flex-col rounded-lg border">
      <div className="flex items-center justify-between gap-3 px-3 py-2">
        <p className="text-muted-foreground min-w-0 truncate text-xs">{totals}</p>
        <Button type="button" size="xs" variant="outline" onClick={() => setOpen(true)}>
          <SlidersHorizontalIcon className="size-3" />
          {chosenCount === 0 ? t("Choose tools") : t("Change tools")}
        </Button>
      </div>

      {core.length > 0 && (
        <div className="border-border flex flex-wrap items-center gap-1.5 border-t px-3 py-2">
          <span className="text-muted-foreground mr-1 shrink-0 text-xs font-medium">
            {t("Always on")}
          </span>
          <ul aria-label={t("Always on")} className="contents">
            {core.map((tool) => (
              <li
                key={tool.name}
                title={tool.description}
                className="bg-sunken text-muted-foreground inline-flex h-6 max-w-full items-center rounded-full px-2 text-xs"
              >
                <span className="truncate">{toolTitle(tool)}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* The chosen list is windowed under a ceiling of its own. An agent
          holding the whole catalog would otherwise mount fifty chips and run
          the height of the panel, which is the thing the dialog exists to
          prevent; the totals and the button above stay put. */}
      {rows.length > 0 && (
        <VirtualRows
          rows={rows}
          estimateSize={40}
          initialHeight={224}
          aria-label={t("Chosen tools")}
          className="border-border max-h-56 border-t"
          rowClassName="border-border border-b last:border-b-0"
        />
      )}

      <ToolPickerDialog
        open={open}
        onOpenChange={setOpen}
        tools={selectable}
        selected={selected}
        tiers={tiers}
        ceiling={ceiling}
        onSelectedChange={onSelectedChange}
        onTiersChange={onTiersChange}
      />
    </div>
  );
}
