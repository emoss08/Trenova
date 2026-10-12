import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { WorkingRun } from "@/lib/graphql/ai-control";
import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { runStatusLabel } from "../activity/agent-badges";
import { Ic } from "../kit/ic";
import { SecH } from "../kit/layout";
import { Tile } from "../kit/marks";

/** A run is short; a quarter minute keeps "working now" honest without a stream. */
const WORKING_REFRESH_MS = 15_000;

type AgentsAtWorkProps = {
  /** Why nothing can run, when something stops every agent. */
  idleReason: "no-provider" | "paused" | null;
  /** Opens one agent on the Agents tab. */
  onOpenAgent: (agentId: string) => void;
  /** Opens the whole roster. */
  onOpenAgents: () => void;
};

/**
 * The agents at a glance: who is working right now and on what, then every agent as a
 * tile, greyed when off, outlined when in shadow, with what waits on a person.
 */
export function AgentsAtWork({ idleReason, onOpenAgent, onOpenAgents }: AgentsAtWorkProps) {
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
    <section className="sec">
      <SecH
        t={t("Agents")}
        n={agentsQuery.isLoading ? null : t("{0} of {1} on", on.length, agents.length)}
        r={
          <button type="button" className="lnk" onClick={onOpenAgents}>
            {t("Roster")}
            <Ic n="arrowR" s={11} />
          </button>
        }
      />
      {working.length > 0 ? (
        <div className="wn">
          {working.map(({ agent, run }) => (
            <button
              key={run.id}
              type="button"
              className="wn-r"
              onClick={() => onOpenAgent(agent.id)}
            >
              <span className="pc-mk">
                <Tile agent={agent} s={26} />
                <i className="pc-dot run sm" />
              </span>
              <span className="wn-t">
                <b>{agent.name}</b>
                <span className="shm">{`${run.summary.trim() || runStatusLabel(run.status, t)}…`}</span>
              </span>
            </button>
          ))}
        </div>
      ) : (
        <p className="wn-e">
          {idleReason === "no-provider"
            ? t("Nothing can run until a provider is connected.")
            : idleReason === "paused"
              ? t("Paused. Nothing is running right now.")
              : t("Nothing is running right now.")}
        </p>
      )}
      <div className="ms">
        {agents.map((agent) => (
          <button
            key={agent.id}
            type="button"
            className={cn("ms-i", !agent.enabled && "off", agent.shadowMode && "sh")}
            title={
              !agent.enabled
                ? t("{0} · off", agent.name)
                : agent.shadowMode
                  ? t("{0} · shadow", agent.name)
                  : agent.name
            }
            onClick={() => onOpenAgent(agent.id)}
          >
            <Tile agent={agent} s={28} />
            {agent.enabled && agent.pendingProposals > 0 && (
              <i className="ms-b mono">{agent.pendingProposals}</i>
            )}
          </button>
        ))}
      </div>
      <div className="lgd">
        <span>
          <i className="k" />
          {t("{0} live", on.length - shadow)}
        </span>
        <span>
          <i className="s" />
          {t("{0} shadow", shadow)}
        </span>
        <span>
          <i className="o" />
          {t("{0} off", agents.length - on.length)}
        </span>
      </div>
    </section>
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
