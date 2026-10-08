import { Dialog } from "@base-ui/react/dialog";
import type { AgentEgressClass, AgentToolPolicy } from "@/lib/graphql/agent-safety";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState, type CSSProperties, type MouseEvent } from "react";
import { Callout, Sel } from "../../edit/fields";
import { Ic } from "../../kit/ic";
import { Seg } from "../../kit/layout";
import { tierWithin } from "../agent-form-schema";
import {
  TIER_ORDER,
  groupToolsByResource,
  impliedReads,
  splitCoreTools,
  toggleTool,
  toolTitle,
} from "../tool-catalog";
import { useDraftField } from "./block";
import { aboveCeiling, capToCeiling, changeTiers, startingTier } from "./builder-model";

/** How far each class of work reaches, nearest first. */
const EGRESS_ORDER: readonly AgentEgressClass[] = [
  "None",
  "Personal",
  "Internal",
  "CustomerVisible",
  "DriverVisible",
  "ExternalRecipient",
  "Money",
];

/** Each class is a category, so it takes an accent hue rather than a tone. */
const EGRESS_DOT: Record<AgentEgressClass, { hue: string; chroma: number }> = {
  None: { hue: "var(--hue-neutral)", chroma: 0.02 },
  Personal: { hue: "var(--hue-teal)", chroma: 0.09 },
  Internal: { hue: "var(--hue-violet)", chroma: 0.11 },
  CustomerVisible: { hue: "var(--hue-sky)", chroma: 0.11 },
  DriverVisible: { hue: "var(--hue-emerald)", chroma: 0.11 },
  ExternalRecipient: { hue: "var(--hue-indigo)", chroma: 0.13 },
  Money: { hue: "var(--hue-amber)", chroma: 0.13 },
};

const DECISION_TIMEOUTS = [3600, 14_400, 86_400, 259_200, 604_800] as const;

/** The furthest any of a tool's work reaches. */
function furthest(rule: AgentToolPolicy | undefined): AgentEgressClass {
  if (!rule) return "None";
  return rule.egress.reduce<AgentEgressClass>(
    (far, egress) => (EGRESS_ORDER.indexOf(egress) > EGRESS_ORDER.indexOf(far) ? egress : far),
    "None",
  );
}

function egressLabel(egress: AgentEgressClass, t: TranslateFn): string {
  switch (egress) {
    case "None":
      return t("Reads only");
    case "Personal":
      return t("Own records");
    case "Internal":
      return t("Internal");
    case "CustomerVisible":
      return t("Customer");
    case "DriverVisible":
      return t("Driver");
    case "ExternalRecipient":
      return t("Outside recipient");
    case "Money":
      return t("Money");
  }
}

export function tierName(tier: AutonomyTier, t: TranslateFn): string {
  switch (tier) {
    case "Propose":
      return t("Propose");
    case "ActWithApproval":
      return t("Ask first");
    case "AutoExecute":
      return t("Automatic");
  }
}

function Dot({ egress, label }: { egress: AgentEgressClass; label: string }) {
  const dot = EGRESS_DOT[egress];
  return (
    <span
      className="eg-d"
      title={label}
      style={{ "--h": dot.hue, "--c": dot.chroma } as CSSProperties}
    />
  );
}

type ToolBenchProps = {
  catalog: readonly ToolCatalogEntry[];
  rules: ReadonlyMap<string, AgentToolPolicy>;
};

/** How many tools the note above the bench names before it counts the rest. */
const NAMED_ABOVE_CEILING = 3;

/**
 * What the agent can read and change, how much freedom each change gets, and the limits
 * that hold however each tool is set: the ceiling, data access and how long a proposal
 * waits.
 */
export function ToolBench({ catalog, rules }: ToolBenchProps) {
  const t = useT();
  const [toolNames, setToolNames] = useDraftField("toolNames");
  const [toolTiers, setToolTiers] = useDraftField("toolTiers");
  const [limits, setLimits] = useDraftField("toolDailyLimits");
  const [ceiling, setCeiling] = useDraftField("autonomyCeiling");
  const [dataAccess, setDataAccess] = useDraftField("dataAccessCeiling");
  const [timeout, setTimeoutSeconds] = useDraftField("decisionTimeoutSeconds");
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<"all" | "act" | "read">("all");
  const [picking, setPicking] = useState(false);
  const [alsoOpen, setAlsoOpen] = useState(false);

  const { core, selectable } = useMemo(() => splitCoreTools(catalog), [catalog]);
  const held = useMemo(() => {
    const chosen = new Set(toolNames);
    return selectable.filter((tool) => chosen.has(tool.name));
  }, [selectable, toolNames]);
  const changes = held.filter((tool) => tool.kind === "action");
  const reads = held.filter((tool) => tool.kind === "query");
  const visible = held.filter((tool) =>
    filter === "act" ? tool.kind === "action" : filter === "read" ? tool.kind === "query" : true,
  );
  const groups = useMemo(
    () => groupToolsByResource(visible, toolNames, query),
    [query, toolNames, visible],
  );
  const implied = useMemo(() => impliedReads(toolNames, catalog), [catalog, toolNames]);
  const { byTier } = changeTiers({ toolNames, toolTiers, autonomyCeiling: ceiling }, catalog);
  const counts = TIER_ORDER.map((tier) => byTier[tier]);
  const over = aboveCeiling({ toolNames, toolTiers, autonomyCeiling: ceiling }, catalog);
  const overTitles = over.tools.map((name) => {
    const tool = catalog.find((entry) => entry.name === name);
    return tool ? toolTitle(tool) : name;
  });

  const maxTier = (tool: ToolCatalogEntry): AutonomyTier =>
    (rules.get(tool.name)?.promotableTier as AutonomyTier | undefined) ?? "AutoExecute";
  const remove = (name: string) => {
    const next = toggleTool(toolNames, toolTiers, name, false);
    setToolNames(next.selected);
    setToolTiers(next.tiers);
  };
  const setLimit = (name: string, raw: string) => {
    const digits = raw.replace(/\D/g, "");
    setLimits({ ...limits, [name]: digits === "" ? null : Math.min(10_000, Number(digits)) });
  };

  const ceilingNote: Record<AutonomyTier, string> = {
    Propose: t("Every change is a proposal"),
    ActWithApproval: t("Changes run once a person approves"),
    AutoExecute: t("Changes may run without asking"),
  };
  const timeouts: Record<(typeof DECISION_TIMEOUTS)[number], string> = {
    3600: t("1 hour"),
    14_400: t("4 hours"),
    86_400: t("1 day"),
    259_200: t("3 days"),
    604_800: t("7 days"),
  };

  return (
    <>
      {over.highest && (
        <Callout
          tone="w"
          action={
            <span className="cal-a">
              <button
                type="button"
                className="btn sm ink"
                onClick={() => setCeiling(over.highest!)}
              >
                {t("Raise the ceiling to {0}", tierName(over.highest, t))}
              </button>
              <button
                type="button"
                className="btn sm"
                onClick={() => setToolTiers(capToCeiling(toolTiers, ceiling))}
              >
                {t("Hold them at {0}", tierName(ceiling, t))}
              </button>
            </span>
          }
        >
          {over.tools.length === 1
            ? t(
                "{0} is set above the ceiling, so it runs as {1}.",
                overTitles[0],
                tierName(ceiling, t),
              )
            : overTitles.length <= NAMED_ABOVE_CEILING
              ? t(
                  "{0} tools are set above the ceiling, so they run as {1}: {2}.",
                  over.tools.length,
                  tierName(ceiling, t),
                  overTitles.join(", "),
                )
              : t(
                  "{0} tools are set above the ceiling, so they run as {1}: {2} and {3} more.",
                  over.tools.length,
                  tierName(ceiling, t),
                  overTitles.slice(0, NAMED_ABOVE_CEILING).join(", "),
                  overTitles.length - NAMED_ABOVE_CEILING,
                )}
        </Callout>
      )}
      <div className="bench">
        <div className="bench-l">
          <div className="bench-tb">
            <button type="button" className="btn sm" onClick={() => setPicking(true)}>
              <Ic n="plus" s={12} />
              {t("Add tools")}
            </button>
            <label className="srch">
              <Ic n="search" s={13} />
              <input
                value={query}
                aria-label={t("Filter this agent's tools")}
                placeholder={t("Filter this agent's tools")}
                onChange={(event) => setQuery(event.target.value)}
              />
              {query && (
                <button
                  type="button"
                  className="ib xs"
                  aria-label={t("Clear the filter")}
                  onClick={() => setQuery("")}
                >
                  <Ic n="x" s={11} />
                </button>
              )}
            </label>
            <div className="seg sm" role="radiogroup" aria-label={t("Show")}>
              {(
                [
                  ["all", t("All"), held.length],
                  ["act", t("Changes"), changes.length],
                  ["read", t("Reads"), reads.length],
                ] as const
              ).map(([key, label, count]) => (
                <button
                  key={key}
                  type="button"
                  role="radio"
                  aria-checked={filter === key}
                  className={filter === key ? "on" : undefined}
                  onClick={() => setFilter(key)}
                >
                  {label}
                  <em className="mono">{count}</em>
                </button>
              ))}
            </div>
          </div>
          <div className="bench-hd" aria-hidden>
            <span>{t("Tool")}</span>
            <span>{t("Freedom")}</span>
            <span>{t("Daily limit")}</span>
            <span />
          </div>
          <div className="bench-list">
            {groups.map((group) => (
              <div key={group.resource} className="bg">
                <div className="bg-h">
                  {group.label}
                  <em className="mono">{group.tools.length}</em>
                </div>
                {[...group.tools]
                  .sort((a, b) =>
                    a.kind === b.kind
                      ? toolTitle(a).localeCompare(toolTitle(b))
                      : a.kind === "action"
                        ? 1
                        : -1,
                  )
                  .map((tool) => {
                    const change = tool.kind === "action";
                    const own = (toolTiers[tool.name] as AutonomyTier | undefined) ?? ceiling;
                    const top = maxTier(tool);
                    const rule = rules.get(tool.name);
                    const egress = furthest(rule);
                    const outside = rule ? rule.readsExternal !== "Never" : false;
                    const title = toolTitle(tool);
                    return (
                      <div key={tool.name} className="br">
                        <span className="br-t">
                          <Dot egress={egress} label={egressLabel(egress, t)} />
                          <span>
                            <b>{title}</b>
                            <em>
                              {change ? egressLabel(egress, t) : t("Reads")}
                              {outside ? ` · ${t("outside text")}` : ""}
                            </em>
                          </span>
                        </span>
                        {change ? (
                          <span
                            className={cn("t3", !tierWithin(own, ceiling) && "capped")}
                            role="radiogroup"
                            aria-label={t("Freedom for {0}", title)}
                          >
                            {TIER_ORDER.map((tier) => {
                              const over = !tierWithin(tier, top);
                              const above = !tierWithin(tier, ceiling);
                              return (
                                <button
                                  key={tier}
                                  type="button"
                                  role="radio"
                                  aria-checked={own === tier}
                                  disabled={over}
                                  className={cn(own === tier && "on", above && "above")}
                                  title={
                                    over
                                      ? t("This tool can't go past {0}", tierName(top, t))
                                      : above
                                        ? t("Above the ceiling — runs as {0}", tierName(ceiling, t))
                                        : undefined
                                  }
                                  onClick={() => setToolTiers({ ...toolTiers, [tool.name]: tier })}
                                >
                                  {over && <Ic n="lock" s={9} />}
                                  {tierName(tier, t)}
                                </button>
                              );
                            })}
                          </span>
                        ) : (
                          <span className="br-rd">{t("Always runs")}</span>
                        )}
                        {change ? (
                          <label className="br-lim">
                            <input
                              className="mono"
                              inputMode="numeric"
                              aria-label={t("Daily limit for {0}", title)}
                              value={limits[tool.name] ? String(limits[tool.name]) : ""}
                              placeholder={t("No limit")}
                              onChange={(event) => setLimit(tool.name, event.target.value)}
                            />
                            <span>{t("/day")}</span>
                          </label>
                        ) : (
                          <span className="br-rd dim">—</span>
                        )}
                        <button
                          type="button"
                          className="ib xs br-x"
                          title={t("Remove {0}", title)}
                          aria-label={t("Remove {0}", title)}
                          onClick={() => remove(tool.name)}
                        >
                          <Ic n="x" s={11} />
                        </button>
                      </div>
                    );
                  })}
              </div>
            ))}
            {held.length === 0 && (
              <div className="bench-e">
                <b>{t("No tools yet")}</b>
                <span>{t("Without tools it can only explain how Trenova works.")}</span>
                <button type="button" className="btn sm ink" onClick={() => setPicking(true)}>
                  <Ic n="plus" s={12} />
                  {t("Add tools")}
                </button>
              </div>
            )}
            {held.length > 0 && groups.length === 0 && (
              <div className="bench-e">
                <span>{t("Nothing matches.")}</span>
              </div>
            )}
          </div>
          <div className={cn("bench-ft", alsoOpen && "open")}>
            <button
              type="button"
              className="bf"
              aria-expanded={alsoOpen}
              onClick={() => setAlsoOpen((open) => !open)}
            >
              <Ic n="chevR" s={11} />
              <span>{t("Also held")}</span>
              <em className="mono">{core.length + implied.length}</em>
              <span className="bf-s">
                {implied.length
                  ? t(
                      "{0} every agent has · {1} that come with your tools",
                      core.length,
                      implied.length,
                    )
                  : t("{0} every agent has", core.length)}
              </span>
            </button>
            {alsoOpen && (
              <div className="bf-g">
                <span className="bf-k">{t("Every agent")}</span>
                <div className="bf-c">
                  {core.map((tool) => (
                    <span key={tool.name} className="bfc">
                      {toolTitle(tool)}
                    </span>
                  ))}
                </div>
                {implied.length > 0 && (
                  <>
                    <span className="bf-k">{t("With your tools")}</span>
                    <div className="bf-c">
                      {implied.map((entry) => {
                        const by = entry.neededBy.map(toolTitle).join(", ");
                        return (
                          <span
                            key={entry.tool.name}
                            className="bfc"
                            title={t("Needed by {0}", by)}
                          >
                            {toolTitle(entry.tool)}
                            <em>{t("for {0}", by)}</em>
                          </span>
                        );
                      })}
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
        <aside className="bench-r">
          <div className="bx">
            <span className="bx-l">{t("Ceiling")}</span>
            <div className="cl" role="radiogroup" aria-label={t("Ceiling")}>
              {TIER_ORDER.map((tier) => (
                <button
                  key={tier}
                  type="button"
                  role="radio"
                  aria-checked={ceiling === tier}
                  className={cn("cl-o", ceiling === tier && "on")}
                  onClick={() => setCeiling(tier)}
                >
                  <span className="cl-r" />
                  <span>
                    <b>{tierName(tier, t)}</b>
                    <em>{ceilingNote[tier]}</em>
                  </span>
                </button>
              ))}
            </div>
            <span className="bx-h">{t("No tool goes past this, whatever it's set to.")}</span>
          </div>
          <div className="bx">
            <span className="bx-l">
              {changes.length === 1
                ? t("How its 1 change runs")
                : t("How its {0} changes run", changes.length)}
            </span>
            <span className="dist">
              {counts.map((count, index) =>
                count ? <i key={index} className={`d${index}`} style={{ flex: count }} /> : null,
              )}
              {changes.length === 0 && <i className="dz0" />}
            </span>
            <div className="dist-l">
              {TIER_ORDER.map((tier, index) => (
                <span key={tier}>
                  <i className={`d${index}`} />
                  {tierName(tier, t)}
                  <b className="mono">{counts[index]}</b>
                </span>
              ))}
            </div>
          </div>
          <div className="bx">
            <span className="bx-l">{t("Data access")}</span>
            <Seg
              v={dataAccess}
              className="sm"
              label={t("Data access")}
              opts={[
                ["Internal", t("Internal")],
                ["Restricted", t("Restricted")],
              ]}
              onChange={setDataAccess}
            />
            <span className="bx-h">
              {dataAccess === "Restricted"
                ? t("May read restricted records like pay and medical files.")
                : t(
                    "Reads internal records only. Pay, medical and other restricted data stay out.",
                  )}
            </span>
          </div>
          <div className="bx">
            <span className="bx-l">{t("Proposals expire after")}</span>
            <Sel
              value={timeout}
              onChange={setTimeoutSeconds}
              label={t("Proposals expire after")}
              options={[
                ...DECISION_TIMEOUTS.map((seconds) => [seconds, timeouts[seconds]] as const),
                ...((DECISION_TIMEOUTS as readonly number[]).includes(timeout)
                  ? []
                  : [[timeout, t("{0} hours", Math.round(timeout / 3600))] as const]),
              ]}
            />
            <span className="bx-h">{t("An undecided proposal is withdrawn after this.")}</span>
          </div>
        </aside>
        <ToolPickerDialog
          open={picking}
          onClose={() => setPicking(false)}
          tools={selectable}
          coreCount={core.length}
          rules={rules}
        />
      </div>
    </>
  );
}

type ToolPickerDialogProps = {
  open: boolean;
  onClose: () => void;
  tools: readonly ToolCatalogEntry[];
  coreCount: number;
  rules: ReadonlyMap<string, AgentToolPolicy>;
};

/** Every tool an agent may be given, grouped by the record it works on. Writes through at once. */
function ToolPickerDialog({ open, onClose, tools, coreCount, rules }: ToolPickerDialogProps) {
  const t = useT();
  const [toolNames, setToolNames] = useDraftField("toolNames");
  const [toolTiers, setToolTiers] = useDraftField("toolTiers");
  const [ceiling] = useDraftField("autonomyCeiling");
  const [query, setQuery] = useState("");
  const [group, setGroup] = useState("all");
  const chosen = new Set(toolNames);
  const allGroups = useMemo(() => groupToolsByResource(tools, toolNames), [toolNames, tools]);
  const shown = useMemo(
    () =>
      groupToolsByResource(tools, toolNames, query).filter(
        (entry) => group === "all" || entry.resource === group,
      ),
    [group, query, toolNames, tools],
  );
  const picked = tools.filter((tool) => chosen.has(tool.name)).length;

  const toggle = (tool: ToolCatalogEntry) => {
    const on = !chosen.has(tool.name);
    const next = toggleTool(toolNames, toolTiers, tool.name, on);
    setToolNames(next.selected);
    if (on && tool.kind === "action") {
      const most = rules.get(tool.name)?.promotableTier as AutonomyTier | undefined;
      setToolTiers({ ...next.tiers, [tool.name]: startingTier(most ?? "AutoExecute", ceiling) });
    } else {
      setToolTiers(next.tiers);
    }
  };

  const onBackdrop = (event: MouseEvent<HTMLDivElement>) => {
    if (event.target === event.currentTarget) onClose();
  };

  return (
    <Dialog.Root open={open} onOpenChange={(next) => !next && onClose()}>
      <Dialog.Portal>
        <div className="aic aic-layer">
          <Dialog.Popup className="tpk-x" onMouseDown={onBackdrop}>
            <div className="tpk">
              <header className="tpk-h">
                <div>
                  <Dialog.Title render={<b />}>{t("Tools")}</Dialog.Title>
                  <Dialog.Description render={<span />}>
                    {t("Pick what this agent can read and change. Tiers are set on the agent.")}
                  </Dialog.Description>
                </div>
                <button type="button" className="ib" aria-label={t("Close")} onClick={onClose}>
                  <Ic n="x" s={14} />
                </button>
              </header>
              <div className="tpk-s">
                <label className="srch">
                  <Ic n="search" s={13} />
                  <input
                    autoFocus
                    value={query}
                    aria-label={t("Search tools")}
                    placeholder={t("Search {0} tools", tools.length)}
                    onChange={(event) => setQuery(event.target.value)}
                  />
                </label>
              </div>
              <div className="tpk-m">
                <nav className="tpk-n" aria-label={t("Records")}>
                  <button
                    type="button"
                    className={group === "all" ? "on" : undefined}
                    onClick={() => setGroup("all")}
                  >
                    <span>{t("All tools")}</span>
                    <em className="mono">{picked}</em>
                  </button>
                  {allGroups.map((entry) => (
                    <button
                      key={entry.resource}
                      type="button"
                      className={group === entry.resource ? "on" : undefined}
                      onClick={() => setGroup(entry.resource)}
                    >
                      <span>{entry.label}</span>
                      {entry.chosen > 0 && <em className="mono">{entry.chosen}</em>}
                    </button>
                  ))}
                </nav>
                <div className="tpk-l">
                  {shown.map((entry) => (
                    <div key={entry.resource} className="tpk-g">
                      <div className="tpk-gh">{entry.label}</div>
                      {[...entry.tools]
                        .sort((a, b) => (a.kind === b.kind ? 0 : a.kind === "action" ? 1 : -1))
                        .map((tool) => {
                          const on = chosen.has(tool.name);
                          const rule = rules.get(tool.name);
                          const top =
                            (rule?.promotableTier as AutonomyTier | undefined) ?? "AutoExecute";
                          return (
                            <button
                              key={tool.name}
                              type="button"
                              role="checkbox"
                              aria-checked={on}
                              className={cn("tpk-r", on && "on")}
                              onClick={() => toggle(tool)}
                            >
                              <span className="tpk-c">{on && <Ic n="check" s={11} w={3} />}</span>
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
                                  {rule && rule.readsExternal !== "Never"
                                    ? ` · ${t("returns outside text")}`
                                    : ""}
                                </em>
                              </span>
                              <span className="mono tpk-nm">{tool.name}</span>
                            </button>
                          );
                        })}
                    </div>
                  ))}
                  {shown.length === 0 && (
                    <div className="nil">{t("No tools match “{0}”.", query)}</div>
                  )}
                </div>
              </div>
              <footer className="tpk-f">
                <span>
                  <b className="mono">{picked}</b> {t("chosen · {0} always on", coreCount)}
                </span>
                <span className="sp" />
                <button type="button" className="btn ink" onClick={onClose}>
                  {t("Done")}
                </button>
              </footer>
            </div>
          </Dialog.Popup>
        </div>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
