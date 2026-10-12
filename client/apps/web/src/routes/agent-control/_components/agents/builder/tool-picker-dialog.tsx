"use no memo";
import { Dialog } from "@base-ui/react/dialog";
import type { AgentToolPolicy } from "@/lib/graphql/agent-safety";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useVirtualizer } from "@tanstack/react-virtual";
import { useCallback, useMemo, useRef, useState, type MouseEvent } from "react";
import { aicFieldTrigger } from "../../edit/field-trigger";
import { Ic } from "../../kit/ic";
import { Seg } from "../../kit/layout";
import { groupToolsByResource, toolTitle } from "../tool-catalog";
import { useDraftField } from "./block";
import { startingTier } from "./builder-model";
import {
  checkState,
  egressLabel,
  furthest,
  pickerGroups,
  pickerRows,
  setTools,
  tierName,
  type CheckState,
  type KindFilter,
  type PickerRow,
} from "./tool-picker-model";
import { Button } from "@trenova/shared/components/ui/button";

const ROW_HEIGHT = { group: 38, tool: 50 } as const;

type ToolPickerDialogProps = {
  open: boolean;
  onClose: () => void;
  tools: readonly ToolCatalogEntry[];
  coreCount: number;
  rules: ReadonlyMap<string, AgentToolPolicy>;
};

/**
 * Every tool an agent may be given, grouped by the record it works on, in one virtual
 * list. Narrow it by search, record, kind or what is already chosen, then take or drop
 * everything shown, a group at a time, or one by one. Writes through to the draft at once.
 */
export function ToolPickerDialog({
  open,
  onClose,
  tools,
  coreCount,
  rules,
}: ToolPickerDialogProps) {
  const t = useT();
  const [toolNames, setToolNames] = useDraftField("toolNames");
  const [toolTiers, setToolTiers] = useDraftField("toolTiers");
  const [ceiling] = useDraftField("autonomyCeiling");
  const [query, setQuery] = useState("");
  const [group, setGroup] = useState("all");
  const [kind, setKind] = useState<KindFilter>("all");
  const [chosenOnly, setChosenOnly] = useState(false);
  const listRef = useRef<HTMLDivElement>(null);

  const chosen = useMemo(() => new Set(toolNames), [toolNames]);
  const rail = useMemo(() => groupToolsByResource(tools, toolNames), [toolNames, tools]);
  const groups = useMemo(
    () => pickerGroups(tools, toolNames, { query, group, kind, chosenOnly }),
    [chosenOnly, group, kind, query, toolNames, tools],
  );
  const rows = useMemo(() => pickerRows(groups, chosen), [chosen, groups]);
  const shown = useMemo(() => groups.flatMap((entry) => entry.tools), [groups]);
  const shownState = checkState(
    shown.map((tool) => tool.name),
    chosen,
  );
  const picked = useMemo(
    () => tools.reduce((count, tool) => (chosen.has(tool.name) ? count + 1 : count), 0),
    [chosen, tools],
  );

  const virtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => listRef.current,
    estimateSize: (index) => ROW_HEIGHT[rows[index]!.kind],
    getItemKey: (index) => rows[index]!.key,
    overscan: 10,
    initialRect: { width: 560, height: 480 },
  });

  const tierFor = useCallback(
    (tool: ToolCatalogEntry) =>
      startingTier(
        (rules.get(tool.name)?.promotableTier as AutonomyTier | undefined) ?? "AutoExecute",
        ceiling,
      ),
    [ceiling, rules],
  );
  const apply = useCallback(
    (subset: readonly ToolCatalogEntry[], on: boolean) => {
      const next = setTools(toolNames, toolTiers, subset, on, tierFor);
      setToolNames(next.selected);
      setToolTiers(next.tiers);
    },
    [setToolNames, setToolTiers, tierFor, toolNames, toolTiers],
  );

  const pickGroup = (resource: string) => {
    setGroup(resource);
    listRef.current?.scrollTo({ top: 0 });
  };
  const onBackdrop = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) onClose();
  };
  const filtered = query.trim() !== "" || group !== "all" || kind !== "all" || chosenOnly;

  return (
    <Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
      <Dialog.Portal>
        <div className="aic aic-layer">
          <Dialog.Popup className="tpk-x" onMouseDown={onBackdrop}>
            <div className="tpk">
              <header className="tpk-h">
                <div>
                  <Dialog.Title render={<b />}>{t("Add tools")}</Dialog.Title>
                  <Dialog.Description render={<span />}>
                    {t("Pick what this agent can read and change. Tiers are set on the agent.")}
                  </Dialog.Description>
                </div>
                <Button type="button" variant="ghost" size="icon-sm" className="text-muted-foreground hover:text-foreground" aria-label={t("Close")} onClick={onClose}>
                  <Ic n="x" s={14} />
                </Button>
              </header>
              <div className="tpk-s">
                <Input
                  autoFocus
                  value={query}
                  aria-label={t("Search tools")}
                  placeholder={t("Search {0} tools", tools.length)}
                  className={aicFieldTrigger}
                  leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
                  rightElement={
                    query ? (
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-xs"
                        className="text-muted-foreground hover:text-foreground"
                        aria-label={t("Clear the search")}
                        onClick={() => setQuery("")}
                      >
                        <XCloseIcon className="size-3" />
                      </Button>
                    ) : undefined
                  }
                  onChange={(event) => setQuery(event.target.value)}
                />
                <div className="tpk-bar">
                  <Seg
                    v={kind}
                    className="sm"
                    label={t("Show")}
                    opts={[
                      ["all", t("All")],
                      ["read", t("Reads")],
                      ["act", t("Changes")],
                    ]}
                    onChange={setKind}
                  />
                  <button
                    type="button"
                    className={cn("tkb", chosenOnly && "on")}
                    aria-pressed={chosenOnly}
                    onClick={() => setChosenOnly((only) => !only)}
                  >
                    <Ic n={chosenOnly ? "check" : "filter"} s={11} w={2.2} />
                    {t("Chosen only")}
                  </button>
                  <span className="sp" />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={shown.length === 0 || shownState === "all"}
                    onClick={() => apply(shown, true)}
                  >
                    <Ic n="check" s={12} />
                    {filtered
                      ? t("Select {0} shown", shown.length)
                      : t("Select all {0}", shown.length)}
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={shownState === "none"}
                    onClick={() => apply(shown, false)}
                  >
                    {filtered ? t("Clear shown") : t("Clear all")}
                  </Button>
                </div>
              </div>
              <div className="tpk-m">
                <ScrollArea
                  className="border-border-subtle min-h-0 border-r max-[1000px]:hidden"
                  maskVariant="popover"
                >
                  <nav className="tpk-n" aria-label={t("Records")}>
                  <button
                    type="button"
                    className={group === "all" ? "on" : undefined}
                    onClick={() => pickGroup("all")}
                  >
                    <span>{t("All tools")}</span>
                    {picked > 0 && <em className="mono">{picked}</em>}
                  </button>
                  {rail.map((entry) => (
                    <button
                      key={entry.resource}
                      type="button"
                      className={group === entry.resource ? "on" : undefined}
                      onClick={() => pickGroup(entry.resource)}
                    >
                      <span>{entry.label}</span>
                      {entry.chosen > 0 && <em className="mono">{entry.chosen}</em>}
                    </button>
                  ))}
                  </nav>
                </ScrollArea>
                <div ref={listRef} className="tpk-l tpk-v">
                  {rows.length > 0 ? (
                    <div className="tpk-vs" style={{ height: virtualizer.getTotalSize() }}>
                      {virtualizer.getVirtualItems().map((item) => {
                        const row = rows[item.index]!;
                        return (
                          <div
                            key={item.key}
                            className="tpk-vr"
                            style={{ height: item.size, transform: `translateY(${item.start}px)` }}
                          >
                            <Row
                              row={row}
                              on={row.kind === "tool" && chosen.has(row.tool.name)}
                              rule={row.kind === "tool" ? rules.get(row.tool.name) : undefined}
                              onToggle={apply}
                            />
                          </div>
                        );
                      })}
                    </div>
                  ) : (
                    <div className="nil">
                      {query.trim()
                        ? t("No tools match “{0}”.", query.trim())
                        : chosenOnly
                          ? t("None chosen here yet.")
                          : t("No tools here.")}
                    </div>
                  )}
                </div>
              </div>
              <footer className="tpk-f">
                <span>
                  <b className="mono">{picked}</b> {t("chosen · {0} always on", coreCount)}
                </span>
                <span className="sp" />
                <Button type="button" variant="default" onClick={onClose}>
                  {t("Done")}
                </Button>
              </footer>
            </div>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

type RowProps = {
  row: PickerRow;
  on: boolean;
  rule: AgentToolPolicy | undefined;
  onToggle: (tools: readonly ToolCatalogEntry[], on: boolean) => void;
};

function Row({ row, on, rule, onToggle }: RowProps) {
  const t = useT();

  if (row.kind === "group") {
    return (
      <button
        type="button"
        role="checkbox"
        aria-checked={ariaChecked(row.state)}
        aria-label={
          row.state === "all"
            ? t("Clear the {0} {1} tools shown", row.tools.length, row.label)
            : t("Select the {0} {1} tools shown", row.tools.length, row.label)
        }
        className={cn("tpk-gh", row.state !== "none" && "on")}
        onClick={() => onToggle(row.tools, row.state !== "all")}
      >
        <Box state={row.state} />
        <span>{row.label}</span>
        <em className="mono">{row.tools.length}</em>
      </button>
    );
  }

  const { tool } = row;
  const top = (rule?.promotableTier as AutonomyTier | undefined) ?? "AutoExecute";
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={on}
      className={cn("tpk-r", on && "on")}
      onClick={() => onToggle([tool], !on)}
    >
      <Box state={on ? "all" : "none"} />
      <span className="tpk-t">
        <b>{toolTitle(tool)}</b>
        <em>
          {tool.kind === "action"
            ? t(
                "Changes · {0} · up to {1}",
                egressLabel(furthest(rule), t),
                tierName(top, t).toLowerCase(),
              )
            : t("Reads")}
          {rule && rule.readsExternal !== "Never" ? ` · ${t("returns outside text")}` : ""}
        </em>
      </span>
      <span className="mono tpk-nm">{tool.name}</span>
    </button>
  );
}

function ariaChecked(state: CheckState): boolean | "mixed" {
  return state === "some" ? "mixed" : state === "all";
}

function Box({ state }: { state: CheckState }) {
  return (
    <span className={cn("tpk-c", state !== "none" && "on")} aria-hidden>
      {state === "all" && <Ic n="check" s={11} w={3} />}
      {state === "some" && <i className="tpk-mx" />}
    </span>
  );
}
