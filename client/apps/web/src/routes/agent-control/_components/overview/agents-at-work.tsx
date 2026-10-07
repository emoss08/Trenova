import { AgentTile } from "@/components/agent-identity/agent-tile";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { ArrowRightIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { runStatusLabel } from "../activity/agent-badges";

/** A run is short; a quarter minute keeps "working now" honest without a stream. */
const WORKING_REFRESH_MS = 15_000;

type AgentsAtWorkProps = {
  /** Why nothing can run, when something stops every agent. */
  idleReason: "no-provider" | "paused" | null;
  onOpenAgents: () => void;
};

/**
 * The agents at a glance: who is working right now and on what, then every agent as a
 * tile, greyed when off, outlined when in shadow, with what waits on a person.
 */
export function AgentsAtWork({ idleReason, onOpenAgents }: AgentsAtWorkProps) {
  const t = useT();
  const agentsQuery = useQuery(queries.assistant.agents(false));
  const runsQuery = useQuery({
    ...queries.aiControl.workingRuns(),
    refetchInterval: idleReason ? false : WORKING_REFRESH_MS,
  });
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const working = useMemo(
    () => workingAgents(agents, runsQuery.data ?? []),
    [agents, runsQuery.data],
  );

  const on = agents.filter((agent) => agent.enabled);
  const shadow = on.filter((agent) => agent.shadowMode).length;

  return (
    <SectionPanel
      title={t("Agents")}
      hint={t("{0} of {1} on", on.length, agents.length)}
      action={
        <Button variant="ghost" size="xs" onClick={onOpenAgents}>
          {t("Roster")}
          <ArrowRightIcon className="size-3" />
        </Button>
      }
    >
      {agentsQuery.isLoading ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-8" />
          <Skeleton className="h-16" />
        </div>
      ) : agents.length === 0 ? (
        <SectionPanelQuiet>{t("No agents yet. Build one from the roster.")}</SectionPanelQuiet>
      ) : (
        <div className="flex flex-col gap-3 p-3">
          {working.length > 0 ? (
            <ul className="flex flex-col gap-1">
              {working.map(({ agent, run }) => (
                <li key={run.id}>
                  <button
                    type="button"
                    onClick={onOpenAgents}
                    className="ui-focus-ring flex w-full items-center gap-2.5 rounded-control px-1.5 py-1 text-left hover:bg-muted"
                  >
                    <span className="relative shrink-0">
                      <AgentTile agent={agent} size="md" />
                      <span className="absolute -right-0.5 -bottom-0.5 size-2 rounded-full border-2 border-card bg-success motion-safe:animate-pulse" />
                    </span>
                    <span className="flex min-w-0 flex-col">
                      <span className="truncate text-sm font-medium">{agent.name}</span>
                      <TextShimmer as="span" className="truncate text-xs">
                        {`${run.summary.trim() || runStatusLabel(run.status, t)}…`}
                      </TextShimmer>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-xs text-muted-foreground">
              {idleReason === "no-provider"
                ? t("Nothing can run until a provider is connected.")
                : idleReason === "paused"
                  ? t("Paused. Nothing is running right now.")
                  : t("Nothing is running right now.")}
            </p>
          )}

          <ul className="flex flex-wrap gap-1.5" aria-label={t("Every agent")}>
            {agents.map((agent) => (
              <li key={agent.id} className="relative">
                <button
                  type="button"
                  onClick={onOpenAgents}
                  title={
                    !agent.enabled
                      ? t("{0} · off", agent.name)
                      : agent.shadowMode
                        ? t("{0} · shadow", agent.name)
                        : agent.name
                  }
                  className={cn(
                    "ui-focus-ring block rounded-md",
                    agent.enabled && agent.shadowMode && "outline-1 outline-offset-1 outline-dashed outline-border-strong",
                  )}
                >
                  <AgentTile
                    agent={agent}
                    size="md"
                    className={cn(!agent.enabled && "opacity-40 grayscale")}
                  />
                </button>
                {agent.enabled && agent.pendingProposals > 0 && (
                  <span className="pointer-events-none absolute -top-1.5 -right-1.5 min-w-4 rounded-full bg-warning px-1 text-center text-2xs font-medium text-warning-on-solid tabular-nums">
                    {agent.pendingProposals}
                  </span>
                )}
              </li>
            ))}
          </ul>

          <div className="flex gap-3 text-xs text-muted-foreground">
            <Legend className="bg-success" label={t("{0} live", on.length - shadow)} />
            <Legend className="border border-dashed border-border-strong" label={t("{0} shadow", shadow)} />
            <Legend className="bg-muted" label={t("{0} off", agents.length - on.length)} />
          </div>
        </div>
      )}
    </SectionPanel>
  );
}

function Legend({ className, label }: { className: string; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span aria-hidden className={cn("size-2 rounded-sm", className)} />
      {label}
    </span>
  );
}

/** Each running agent once, with its newest run. */
export function workingAgents(
  agents: readonly AgentDefinitionRow[],
  runs: readonly WorkingRun[],
): { agent: AgentDefinitionRow; run: WorkingRun }[] {
  const byID = new Map(agents.map((agent) => [agent.id, agent]));
  const seen = new Set<string>();
  const out: { agent: AgentDefinitionRow; run: WorkingRun }[] = [];
  for (const run of runs) {
    const agent = byID.get(run.agentDefinitionId);
    if (!agent || seen.has(agent.id)) {
      continue;
    }
    seen.add(agent.id);
    out.push({ agent, run });
  }
  return out;
}
