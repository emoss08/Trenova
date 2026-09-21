import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { cn } from "@trenova/shared/lib/utils";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { PencilLineIcon, RotateCcwIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { tierWithin } from "./agent-form-schema";
import {
  TIER_LABEL,
  TIER_ORDER,
  groupToolsByResource,
  toggleTool,
  toolTitle,
  type ToolGroup,
} from "./tool-catalog";

export type ToolPickerDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tools: readonly ToolCatalogEntry[];
  selected: string[];
  tiers: Record<string, AutonomyTier>;
  ceiling: AutonomyTier;
  onSelectedChange: (next: string[]) => void;
  onTiersChange: (next: Record<string, AutonomyTier>) => void;
};

/**
 * The whole catalog, in a dialog of its own.
 *
 * Fifty tools in twenty-six groups used to sit open in the middle of the
 * form, the tallest thing in the app. Here they take a two-pane picker: the
 * groups down the left with how many of each the agent holds, the tools of
 * the open group on the right. Reads run when the agent asks; a change
 * carries its own tier, capped by the ceiling, chosen on the row.
 */
export function ToolPickerDialog({
  open,
  onOpenChange,
  tools,
  selected,
  tiers,
  ceiling,
  onSelectedChange,
  onTiersChange,
}: ToolPickerDialogProps) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [resource, setResource] = useState<string | null>(null);

  const groups = useMemo(
    () => groupToolsByResource(tools, selected, query),
    [query, selected, tools],
  );
  const searching = query.trim() !== "";
  // A search shows every match across the groups; otherwise one group is
  // open, the first when none was picked or the picked one has emptied.
  const openGroups: ToolGroup[] = searching
    ? groups
    : groups.filter((group) => group.resource === (resource ?? groups[0]?.resource));
  const selectedSet = useMemo(() => new Set(selected), [selected]);
  const readNames = useMemo(
    () => tools.filter((tool) => tool.kind === "query").map((tool) => tool.name),
    [tools],
  );

  const toggle = (name: string, on: boolean) => {
    const next = toggleTool(selected, tiers, name, on);
    onSelectedChange(next.selected);
    if (next.tiers !== tiers) {
      onTiersChange(next.tiers);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="flex max-h-[85dvh] flex-col gap-0 p-0">
        <DialogHeader className="border-border flex flex-col gap-3 border-b px-5 pt-5 pb-4">
          <div className="flex flex-col gap-1">
            <DialogTitle>{t("Tools")}</DialogTitle>
            <DialogDescription>
              {t(
                "Everything the system offers. Reads run as soon as the agent asks; a change carries its own autonomy, never above the agent's ceiling.",
              )}
            </DialogDescription>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Input
              inputContainerClassName="w-full max-w-xs"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t("Search tools")}
              className="h-8"
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
              aria-label={t("Search tools")}
            />
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} of {1} chosen", selected.length, tools.length)}
            </span>
            <div className="ml-auto flex gap-1.5">
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={() => onSelectedChange([...new Set([...selected, ...readNames])])}
              >
                {t("All reads")}
              </Button>
              <Button
                type="button"
                size="xs"
                variant="ghost"
                disabled={selected.length === 0}
                onClick={() => {
                  onSelectedChange([]);
                  onTiersChange({});
                }}
              >
                {t("Clear")}
              </Button>
            </div>
          </div>
        </DialogHeader>

        <div className="grid min-h-0 flex-1 grid-cols-[11rem_minmax(0,1fr)]">
          <ScrollArea
            className="border-border min-h-0 border-r"
            viewportClassName="max-h-[60dvh]"
            maskVariant="background"
          >
            <nav aria-label={t("Tool groups")} className="flex flex-col gap-0.5 p-2">
              {groups.map((group) => {
                const active = !searching && group.resource === openGroups[0]?.resource;
                return (
                  <button
                    key={group.resource}
                    type="button"
                    aria-current={active ? "true" : undefined}
                    onClick={() => {
                      setResource(group.resource);
                      setQuery("");
                    }}
                    className={cn(
                      "ui-focus-ring flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors",
                      active
                        ? "bg-surface-selected text-foreground"
                        : "text-muted-foreground hover:bg-surface-hover hover:text-foreground",
                    )}
                  >
                    <span className="truncate">{group.label}</span>
                    {group.chosen > 0 && (
                      <span className="text-foreground shrink-0 text-xs tabular-nums">
                        {group.chosen}
                      </span>
                    )}
                  </button>
                );
              })}
            </nav>
          </ScrollArea>

          <ScrollArea
            className="min-h-0"
            viewportClassName="max-h-[60dvh]"
            maskVariant="background"
          >
            {openGroups.length === 0 ? (
              <p className="text-muted-foreground px-5 py-10 text-center text-sm">
                {t("No tools match that search.")}
              </p>
            ) : (
              <div className="flex flex-col gap-4 px-4 py-3">
                {openGroups.map((group) => (
                  <section key={group.resource} className="flex flex-col gap-1">
                    {searching && (
                      <h4 className="text-muted-foreground px-1 text-xs font-medium">
                        {group.label}
                      </h4>
                    )}
                    <ul className="flex flex-col">
                      {group.tools.map((tool) => (
                        <ToolRow
                          key={tool.name}
                          tool={tool}
                          checked={selectedSet.has(tool.name)}
                          tier={tiers[tool.name] ?? ceiling}
                          ceiling={ceiling}
                          onToggle={(on) => toggle(tool.name, on)}
                          onTier={(tier) => onTiersChange({ ...tiers, [tool.name]: tier })}
                        />
                      ))}
                    </ul>
                  </section>
                ))}
              </div>
            )}
          </ScrollArea>
        </div>

        <DialogFooter className="mx-0 mb-0 rounded-b-none">
          <Button type="button" onClick={() => onOpenChange(false)}>
            {t("Done")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ToolRow({
  tool,
  checked,
  tier,
  ceiling,
  onToggle,
  onTier,
}: {
  tool: ToolCatalogEntry;
  checked: boolean;
  tier: AutonomyTier;
  ceiling: AutonomyTier;
  onToggle: (on: boolean) => void;
  onTier: (tier: AutonomyTier) => void;
}) {
  const t = useT();
  const title = toolTitle(tool);
  const id = `tool-${tool.name}`;

  return (
    <li
      className={cn(
        "flex items-start gap-3 rounded-md px-2 py-2 transition-colors",
        checked && "bg-surface-selected",
      )}
    >
      <Checkbox
        id={id}
        checked={checked}
        onCheckedChange={(value) => onToggle(value === true)}
        className="mt-0.5"
      />
      <label htmlFor={id} className="min-w-0 flex-1 cursor-pointer">
        <span className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
          <span className="text-sm font-medium">{title}</span>
          {tool.kind === "action" ? (
            <Badge variant="warning" className="h-4 gap-1 px-1 text-2xs">
              <PencilLineIcon className="size-2.5" />
              {t("Changes data")}
            </Badge>
          ) : null}
          {tool.kind === "action" && tool.reversible && (
            <span className="text-muted-foreground inline-flex items-center gap-0.5 text-2xs">
              <RotateCcwIcon className="size-2.5" />
              {t("Reversible")}
            </span>
          )}
        </span>
        <span className="text-muted-foreground block text-xs">{tool.description}</span>
      </label>
      {checked && tool.kind === "action" && (
        <div
          role="radiogroup"
          aria-label={t("Autonomy for {0}", title)}
          className="bg-sunken flex shrink-0 rounded-md p-0.5"
        >
          {TIER_ORDER.map((option) => {
            const allowed = tierWithin(option, ceiling);
            return (
              <button
                key={option}
                type="button"
                role="radio"
                aria-checked={tier === option}
                disabled={!allowed}
                onClick={() => onTier(option)}
                title={allowed ? undefined : t("Above this agent's ceiling")}
                className={cn(
                  "ui-focus-ring rounded-md px-2 py-0.5 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-40",
                  tier === option
                    ? "bg-card text-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                {t(TIER_LABEL[option])}
              </button>
            );
          })}
        </div>
      )}
    </li>
  );
}
