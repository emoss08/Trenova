import { SegmentedField } from "@/components/fields/segmented-field";
import { SelectField } from "@/components/fields/select-field";
import type { AgentEgressClass, AgentToolPolicy } from "@/lib/graphql/agent-safety";
import type { AutonomyTier, ToolCatalogEntry } from "@/types/assistant";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Input } from "@trenova/shared/components/ui/input";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { lazy, Suspense, useMemo, useState, type CSSProperties } from "react";
import { useFormContext } from "react-hook-form";
import { TOOL_PICKER_PARAM, toolPickerParser } from "../../../ai-control-tabs";
import { useAddressedFlag } from "../../../use-addressed-flag";
import { Callout } from "../../edit/callout";
import { aicFieldTrigger, aicToolbarFieldTrigger } from "../../edit/field-trigger";
import { Ic } from "../../kit/ic";
import { tierWithin, type AgentFormValues } from "../agent-form-schema";
import {
  TIER_ORDER,
  groupToolsByResource,
  impliedReads,
  splitCoreTools,
  toggleTool,
  toolTitle,
} from "../tool-catalog";
import { useDraftField } from "./block";
import { aboveCeiling, capToCeiling, changeTiers } from "./builder-model";
import { egressLabel, furthest, tierName } from "./tool-picker-model";
import { Button } from "@trenova/shared/components/ui/button";

const ToolPickerDialog = lazy(() =>
  import("./tool-picker-dialog").then((module) => ({ default: module.ToolPickerDialog })),
);

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
  const [dataAccess] = useDraftField("dataAccessCeiling");
  const [timeout, setTimeoutSeconds] = useDraftField("decisionTimeoutSeconds");
  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<"all" | "act" | "read">("all");
  const [picking, setPicking] = useAddressedFlag(TOOL_PICKER_PARAM, toolPickerParser);
  const [pickerLoaded, setPickerLoaded] = useState(false);
  const { control } = useFormContext<AgentFormValues>();
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
  const openPicker = () => {
    setPickerLoaded(true);
    setPicking(true);
  };
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
              <Button
                type="button"
                variant="default"
                size="sm"
                onClick={() => setCeiling(over.highest!)}
              >
                {t("Raise the ceiling to {0}", tierName(over.highest, t))}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => setToolTiers(capToCeiling(toolTiers, ceiling))}
              >
                {t("Hold them at {0}", tierName(ceiling, t))}
              </Button>
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
            <Button type="button" variant="outline" size="sm" onClick={openPicker}>
              <Ic n="plus" s={12} />
              {t("Add tools")}
            </Button>
            <Input
              value={query}
              aria-label={t("Filter this agent's tools")}
              placeholder={t("Filter this agent's tools")}
              className={aicToolbarFieldTrigger}
              inputContainerClassName="w-full max-w-80 flex-[0_1_20rem]"
              leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
              rightElement={
                query ? (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    className="text-muted-foreground hover:text-foreground"
                    aria-label={t("Clear the filter")}
                    onClick={() => setQuery("")}
                  >
                    <XCloseIcon className="size-3" />
                  </Button>
                ) : undefined
              }
              onChange={(event) => setQuery(event.target.value)}
            />
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
                          <Input
                            inputMode="numeric"
                            aria-label={t("Daily limit for {0}", title)}
                            value={limits[tool.name] ? String(limits[tool.name]) : ""}
                            placeholder={t("No limit")}
                            sideText={t("/day")}
                            onChange={(event) => setLimit(tool.name, event.target.value)}
                          />
                        ) : (
                          <span className="br-rd dim">—</span>
                        )}
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon-xs"
                          className="text-muted-foreground hover:text-foreground br-x"
                          title={t("Remove {0}", title)}
                          aria-label={t("Remove {0}", title)}
                          onClick={() => remove(tool.name)}
                        >
                          <Ic n="x" s={11} />
                        </Button>
                      </div>
                    );
                  })}
              </div>
            ))}
            {held.length === 0 && (
              <div className="bench-e">
                <b>{t("No tools yet")}</b>
                <span>{t("Without tools it can only explain how Trenova works.")}</span>
                <Button type="button" variant="default" size="sm" onClick={openPicker}>
                  <Ic n="plus" s={12} />
                  {t("Add tools")}
                </Button>
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
            <SegmentedField<AgentFormValues, AgentFormValues["dataAccessCeiling"]>
              control={control}
              name="dataAccessCeiling"
              label={t("Data access")}
              description={
                dataAccess === "Restricted"
                  ? t("May read restricted records like pay and medical files.")
                  : t(
                      "Reads internal records only. Pay, medical and other restricted data stay out.",
                    )
              }
              options={[
                { value: "Internal", label: t("Internal") },
                { value: "Restricted", label: t("Restricted") },
              ]}
            />
          </div>
          <div className="bx">
            <SelectField<AgentFormValues>
              control={control}
              name="decisionTimeoutSeconds"
              rules={{ required: true }}
              label={t("Proposals expire after")}
              description={t("An undecided proposal is withdrawn after this.")}
              triggerClassName={aicFieldTrigger}
              placeholder={t("Proposals expire after")}
              onValueChange={(value) => setTimeoutSeconds(Number(value))}
              options={[
                ...DECISION_TIMEOUTS.map((seconds) => ({
                  value: seconds,
                  label: timeouts[seconds],
                })),
                ...((DECISION_TIMEOUTS as readonly number[]).includes(timeout)
                  ? []
                  : [{ value: timeout, label: t("{0} hours", Math.round(timeout / 3600)) }]),
              ]}
            />
          </div>
        </aside>
        {(pickerLoaded || picking) && (
          <Suspense fallback={null}>
            <ToolPickerDialog
              open={picking}
              onClose={() => setPicking(false)}
              tools={selectable}
              coreCount={core.length}
              rules={rules}
            />
          </Suspense>
        )}
      </div>
    </>
  );
}
