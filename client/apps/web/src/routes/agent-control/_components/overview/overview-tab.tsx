import { usePermission } from "@/hooks/use-permission";
import { agentControlQueryOptions } from "@/lib/graphql/agent-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { POLICY_EDITOR_PARAM, policyEditorParser } from "../../ai-control-tabs";
import { useAddressedFlag } from "../../use-addressed-flag";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { NovaSummary } from "../nova/nova-summary";
import { useNovaTargets } from "../nova/use-nova-targets";
import { PolicyEditor } from "../policy/policy-editor";
import { AgentsAtWork } from "./agents-at-work";
import { OrganizationWide } from "./organization-wide";
import { OVERVIEW_WINDOW_DAYS, OverviewFigures } from "./overview-figures";
import { OverviewControl } from "./overview-control";
import { ProviderFailureStrip } from "./provider-failure-strip";
import { SetupSteps } from "./setup-steps";
import { TuneUps } from "./tune-ups";
import { UsageByFeature } from "./usage-by-feature";

/** The sentence is reread a few seconds after a read that left a model rewording it. */
const PENDING_REFRESH_MS = 3_000;
/** Navigating back within this long shows the sentence already read, without a request. */
const SUMMARY_STALE_MS = 30_000;

type OverviewTabProps = {
  onOpenProviders: () => void;
  onOpenAgents: () => void;
};

/**
 * AI in the organization on one screen: Nova's sentence and the one control beside it, the
 * week in figures, a provider that is failing, where the calls went, and the agents and
 * organization-wide settings in a side column. With no provider, the way to connect one.
 */
export default function OverviewTab({ onOpenProviders, onOpenAgents }: OverviewTabProps) {
  const t = useT();
  const go = useAIControlNavigation();
  const onTarget = useNovaTargets();
  const [editingPolicy, setEditingPolicy] = useAddressedFlag(
    POLICY_EDITOR_PARAM,
    policyEditorParser,
  );
  const { allowed: canUpdateControl } = usePermission(Resource.AgentControl, Operation.Update);

  const summaryQuery = useQuery({
    ...queries.aiControl.summary("Overview"),
    staleTime: SUMMARY_STALE_MS,
    refetchInterval: (query) => (query.state.data?.pending ? PENDING_REFRESH_MS : false),
  });
  const controlQuery = useQuery(agentControlQueryOptions());
  const providersQuery = useQuery(queries.aiProvider.list());
  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const usageQuery = useQuery(queries.aiProvider.usage(OVERVIEW_WINDOW_DAYS));

  const summary = summaryQuery.data;
  const facts = summary?.facts;
  const noProvider = facts ? facts.providersOn === 0 : false;

  const editProvider = useCallback(
    (providerId: string) => go({ tab: "providers", panel: { mode: "edit", entityId: providerId } }),
    [go],
  );
  const openAgent = useCallback((agentId: string) => go({ tab: "agents", agent: agentId }), [go]);
  const firstEnabled = providersQuery.data?.find((provider) => provider.enabled);
  const taskCount = catalogQuery.data?.tasks.length ?? 0;
  const busiest = Math.max(0, ...(usageQuery.data?.byFeature ?? []).map((slice) => slice.calls));

  return (
    <div className="tabp ov2">
      <NovaSummary
        context={t("AI control")}
        segments={summary?.segments}
        loading={summaryQuery.isLoading}
        working={Boolean(facts && facts.agents.working > 0 && !facts.paused && !noProvider)}
        onTarget={onTarget}
        control={
          <OverviewControl
            control={controlQuery.data}
            noProvider={noProvider}
            canUpdate={canUpdateControl}
            onConnectProvider={() => go({ tab: "providers", panel: { mode: "create" } })}
          />
        }
      />

      {noProvider ? (
        <SetupSteps
          agentCount={facts?.agents.total ?? 0}
          onPickPreset={(preset) =>
            go({ tab: "providers", panel: { mode: "create", preset: preset ?? undefined } })
          }
        />
      ) : (
        <OverviewFigures
          onSetPrices={firstEnabled ? () => editProvider(firstEnabled.id) : undefined}
        />
      )}

      {!noProvider && summary && (
        <ProviderFailureStrip failures={summary.visibleFailures} onEditProvider={editProvider} />
      )}

      <div className="ov">
        <div className="ov-m">
          {!noProvider && <TuneUps />}
          {!noProvider && <UsageByFeature days={OVERVIEW_WINDOW_DAYS} busiest={busiest} />}
        </div>
        <aside className="ov-a">
          <AgentsAtWork
            idleReason={noProvider ? "no-provider" : facts?.paused ? "paused" : null}
            onOpenAgent={openAgent}
            onOpenAgents={onOpenAgents}
          />
          {controlQuery.data && (
            <OrganizationWide
              control={controlQuery.data}
              routing={
                noProvider || !facts || taskCount === 0
                  ? null
                  : { covered: Math.max(0, taskCount - facts.uncovered), total: taskCount }
              }
              canEdit={canUpdateControl}
              onEdit={() => setEditingPolicy(true)}
              onOpenRouting={onOpenProviders}
            />
          )}
        </aside>
      </div>

      {controlQuery.data && (
        <PolicyEditor
          open={editingPolicy && canUpdateControl}
          control={controlQuery.data}
          onClose={() => setEditingPolicy(false)}
        />
      )}
    </div>
  );
}
