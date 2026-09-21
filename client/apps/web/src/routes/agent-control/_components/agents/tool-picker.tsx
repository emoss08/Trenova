import { useT } from "@trenova/shared/i18n/use-t";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn, toTitleCase } from "@trenova/shared/lib/utils";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { PencilLineIcon, RotateCcwIcon, SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { tierWithin } from "./agent-form-schema";

type ToolPickerProps = {
  tools: readonly ToolCatalogEntry[];
  isLoading?: boolean;
  selected: string[];
  tiers: Record<string, AutonomyTier>;
  ceiling: AutonomyTier;
  onSelectedChange: (next: string[]) => void;
  onTiersChange: (next: Record<string, AutonomyTier>) => void;
};

const TIER_ORDER: AutonomyTier[] = ["Propose", "ActWithApproval", "AutoExecute"];

const TIER_LABEL: Record<AutonomyTier, string> = {
  Propose: "Propose",
  ActWithApproval: "Ask first",
  AutoExecute: "Automatic",
};

/**
 * Resources whose title-cased name reads wrong. Title case turns an
 * abbreviation into a word — worker_pto becomes "Worker Pto" — and this is the
 * label an administrator reads while deciding what an agent may touch.
 */
const RESOURCE_LABELS: Record<string, string> = {
  worker_pto: "Worker time off",
  hazardous_material: "Hazardous materials",
};

function resourceLabel(resource: string): string {
  return RESOURCE_LABELS[resource] ?? toTitleCase(resource.replace(/[_-]+/g, " "));
}

/**
 * Every tool the system offers, grouped by what it touches. Reads run as
 * soon as the agent asks; changes carry their own tier, capped by the
 * agent's ceiling, so one agent can look anything up and still need a person
 * before it moves a driver.
 */
export function ToolPicker({
  tools,
  isLoading = false,
  selected,
  tiers,
  ceiling,
  onSelectedChange,
  onTiersChange,
}: ToolPickerProps) {
  const t = useT();
  const [query, setQuery] = useState("");

  const groups = useMemo(() => {
    const needle = query.trim().toLowerCase();
    const visible = tools.filter((tool) => {
      if (needle === "") return true;
      const title = describeToolCall(tool.name, null).title.toLowerCase();
      return (
        tool.name.includes(needle) ||
        title.includes(needle) ||
        tool.description.toLowerCase().includes(needle) ||
        tool.resource.toLowerCase().includes(needle)
      );
    });
    const byResource = new Map<string, ToolCatalogEntry[]>();
    for (const tool of visible) {
      const key = tool.resource || "general";
      byResource.set(key, [...(byResource.get(key) ?? []), tool]);
    }
    return [...byResource.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([resource, entries]) => ({
        resource,
        tools: [...entries].sort((a, b) =>
          a.kind === b.kind ? a.name.localeCompare(b.name) : a.kind === "query" ? -1 : 1,
        ),
      }));
  }, [query, tools]);

  const selectedSet = useMemo(() => new Set(selected), [selected]);
  const readNames = useMemo(
    () => tools.filter((tool) => tool.kind === "query").map((tool) => tool.name),
    [tools],
  );

  const toggle = (name: string, checked: boolean) => {
    if (checked) {
      if (!selectedSet.has(name)) onSelectedChange([...selected, name]);
      return;
    }
    onSelectedChange(selected.filter((entry) => entry !== name));
    if (name in tiers) {
      onTiersChange(Object.fromEntries(Object.entries(tiers).filter(([tool]) => tool !== name)));
    }
  };

  const setTier = (name: string, tier: AutonomyTier) => {
    onTiersChange({ ...tiers, [name]: tier });
  };

  if (isLoading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-8" />
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("Search tools")}
          className="h-8 max-w-xs text-xs"
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          aria-label={t("Search tools")}
        />
        <span className="text-muted-foreground text-xs tabular-nums">
          {t("{0} of {1} selected", selected.length, tools.length)}
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

      {groups.length === 0 ? (
        <p className="text-muted-foreground py-6 text-center text-xs">
          {t("No tools match that search.")}
        </p>
      ) : (
        <div className="border-border/70 divide-border/70 divide-y rounded-lg border">
          {groups.map((group) => (
            <section key={group.resource} className="flex flex-col">
              <h4 className="bg-muted/40 text-muted-foreground px-3 py-1.5 text-xs font-medium">
                {resourceLabel(group.resource)}
              </h4>
              <ul className="divide-border/60 divide-y">
                {group.tools.map((tool) => {
                  const checked = selectedSet.has(tool.name);
                  const title = describeToolCall(tool.name, null).title;
                  const tier = tiers[tool.name] ?? ceiling;
                  return (
                    <li
                      key={tool.name}
                      className={cn(
                        "flex items-start gap-3 px-3 py-2 transition-colors",
                        checked && "bg-primary/5",
                      )}
                    >
                      <Checkbox
                        id={`tool-${tool.name}`}
                        checked={checked}
                        onCheckedChange={(value) => toggle(tool.name, value === true)}
                        className="mt-0.5"
                      />
                      <label
                        htmlFor={`tool-${tool.name}`}
                        className="min-w-0 flex-1 cursor-pointer"
                      >
                        <span className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                          <span className="text-sm font-medium">{title}</span>
                          <span className="text-muted-foreground font-mono text-xs">
                            {tool.name}
                          </span>
                          {tool.kind === "query" ? (
                            <Badge variant="neutral" appearance="outline" className="h-4 gap-1 px-1 text-2xs">
                              <SearchIcon className="size-2.5" />
                              {t("Reads")}
                            </Badge>
                          ) : (
                            <Badge variant="warning" className="h-4 gap-1 px-1 text-2xs">
                              <PencilLineIcon className="size-2.5" />
                              {t("Changes data")}
                            </Badge>
                          )}
                          {tool.kind === "action" && tool.reversible && (
                            <span className="text-muted-foreground inline-flex items-center gap-0.5 text-2xs">
                              <RotateCcwIcon className="size-2.5" />
                              {t("Reversible")}
                            </span>
                          )}
                        </span>
                        <span className="text-muted-foreground block text-xs">
                          {tool.description}
                        </span>
                      </label>
                      {checked && tool.kind === "action" && (
                        <div
                          role="radiogroup"
                          aria-label={t("Autonomy for {0}", title)}
                          className="bg-muted flex shrink-0 rounded-md p-0.5"
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
                                onClick={() => setTier(tool.name, option)}
                                title={allowed ? undefined : t("Above this agent's ceiling")}
                                className={cn(
                                  "rounded px-2 py-0.5 text-xs transition-colors disabled:cursor-not-allowed disabled:opacity-40",
                                  tier === option
                                    ? "bg-background text-foreground shadow-xs"
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
                })}
              </ul>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
