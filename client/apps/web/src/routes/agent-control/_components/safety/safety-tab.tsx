import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { AgentEgressClass } from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryStates } from "nuqs";
import { lazy } from "react";
import type { SafetyView } from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Ic } from "../kit/ic";
import { Figs, SecH, Seg } from "../kit/layout";
import { NovaSummary } from "../nova/nova-summary";
import { EgressMap } from "./egress-map";
import { SAFETY_SUMMARY_STALE_MS } from "./safety-figures";
import { SafetyHero } from "./safety-hero";
import { EGRESS_ORDER, REVIEWED_AT_ONCE, safetyFacts } from "./safety-model";

const ToolRulesTable = lazy(() => import("./tool-rules-table"));
const ByAgentView = lazy(() => import("./by-agent"));

const filterParsers = { fieldFilters: searchParamsParser.fieldFilters };

/**
 * What the AI can do without a person, answered from the same policies the runtime
 * decides every call from: Nova's sentence and the review it suggests, three figures,
 * the tools that change things by who sees their work, then one table at a time: every
 * tool's rule, or what the agents someone picks make of the tools they hold.
 */
export default function SafetyTab({ view }: { view: SafetyView }) {
  const t = useT();
  const navigate = useAIControlNavigation();
  const [{ fieldFilters }] = useQueryStates(filterParsers);
  const summary = useQuery({
    ...queries.agentSafety.summary(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });

  const activeEgress =
    view === "rules"
      ? (((fieldFilters ?? []).find((filter) => filter.field === "egress")?.value as
          | AgentEgressClass
          | undefined) ?? null)
      : null;

  const pickEgress = (egress: AgentEgressClass | null) =>
    navigate({
      tab: "safety",
      view: "rules",
      fieldFilters: egress ? [{ field: "egress", operator: "eq", value: egress }] : undefined,
    });

  if (summary.isError) {
    return (
      <div className="tabp">
        <div className="bnr d" role="alert">
          <Ic n="alert" s={14} />
          <span>
            {t("What agents can do without a person could not be loaded. Try again shortly.")}
          </span>
        </div>
      </div>
    );
  }

  const facts = summary.data ? safetyFacts(summary.data) : null;

  return (
    <div className="tabp">
      {facts ? (
        <SafetyHero
          facts={facts}
          onShowRunning={() =>
            navigate({
              tab: "safety",
              view: "rules",
              fieldFilters: [{ field: "runsWithoutPerson", operator: "eq", value: true }],
            })
          }
          onReviewOpen={() =>
            navigate({
              tab: "safety",
              view: "agents",
              safetyAgents: facts.open.slice(0, REVIEWED_AT_ONCE),
            })
          }
        />
      ) : (
        <NovaSummary context={t("Safety")} segments={undefined} loading onTarget={() => {}} />
      )}
      {facts && summary.data && (
        <>
          <Figs
            items={[
              {
                label: t("Tools that run without a person"),
                value: facts.runs,
                tone: facts.runs > 0 ? "t-b" : undefined,
                sub:
                  facts.runs > 0
                    ? t("on at least one agent · {0}", facts.runningTools.join(", "))
                    : t("on at least one agent"),
              },
              {
                label: t("Tools that send outside the organization"),
                value: facts.leave,
                sub: t("never past approval"),
              },
              {
                label: t("Open agents with sensitive tools"),
                value: facts.open.length,
                tone: facts.open.length > 0 ? "t-w" : undefined,
                sub: t("usable by everyone"),
              },
            ]}
          />
          <section className="sec">
            <SecH
              t={t("Who sees the work")}
              n={t(
                "{0, plural, one {# tool that changes things} other {# tools that change things}}",
                summary.data.egressCounts.reduce((sum, entry) => sum + entry.count, 0),
              )}
              r={
                activeEgress ? (
                  <button type="button" className="lnk" onClick={() => pickEgress(null)}>
                    {t("Clear")}
                  </button>
                ) : null
              }
            />
            <EgressMap
              counts={summary.data.egressCounts}
              active={activeEgress && EGRESS_ORDER.includes(activeEgress) ? activeEgress : null}
              onPick={pickEgress}
            />
          </section>
        </>
      )}
      <div className="vw">
        <Seg
          label={t("View")}
          v={view}
          opts={[
            ["rules", t("Tool rules")],
            ["agents", t("By agent")],
          ]}
          onChange={(next) => navigate({ tab: "safety", view: next })}
        />
      </div>
      <DataTableLazyComponent>
        {view === "rules" && <ToolRulesTable />}
        {view === "agents" && <ByAgentView />}
      </DataTableLazyComponent>
    </div>
  );
}
