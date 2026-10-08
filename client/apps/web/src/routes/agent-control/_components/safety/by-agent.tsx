import { DataTable } from "@/components/data-table/data-table";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import {
  AGENT_TOOL_SAFETY_LIST_KEY,
  createAgentToolSafetyTableGraphQLConfig,
  type AgentSafetyHeader,
  type AgentToolSafetyRow,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { createContext, useCallback, useContext, useMemo, useState } from "react";
import {
  CLEARED_TABLE_STATE,
  SAFETY_AGENTS_PARAM,
  safetyAgentsParser,
} from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { Menu } from "../kit/controls";
import { Ic } from "../kit/ic";
import { Tile } from "../kit/marks";
import { getAgentToolColumns, holdersOf, type ToolHolding } from "./safety-columns";
import { SAFETY_SUMMARY_STALE_MS } from "./safety-figures";
import { reachLabel, tierLabel } from "./safety-model";
import { useToolHolding } from "./tool-rules-table";
import { ToolSheet } from "./tool-sheet";

/** An agent's answers move with its settings and trust, so they are re-read after a minute. */
const AGENT_SAFETY_STALE_MS = 60_000;

/** The most agents laid side by side. */
export const MAX_COMPARED_AGENTS = 4;

const NO_RESOURCES: readonly string[] = [];
const NO_HEADERS: readonly AgentSafetyHeader[] = [];
const NO_AGENTS: readonly AgentDefinitionRow[] = [];

const byAgentParsers = {
  ...searchParamsParser,
  [SAFETY_AGENTS_PARAM]: safetyAgentsParser,
};

const HoldingContext = createContext<ToolHolding>({ byTool: new Map(), agents: new Map() });

/**
 * What the agents someone picks can do without a person, tool by tool: once for a run
 * that has read nothing from outside, once for a run that has. Every picked agent's tools
 * are one table, so they page, filter, sort and scroll together; nothing is read until an
 * agent is picked.
 */
export default function ByAgentView() {
  const t = useT();
  const [params, setParams] = useQueryStates(byAgentParsers);
  const stored = params[SAFETY_AGENTS_PARAM];
  const picked = useMemo(() => [...new Set(stored)].slice(0, MAX_COMPARED_AGENTS), [stored]);
  const agentsQuery = useQuery(queries.assistant.agents(false));
  const agents = agentsQuery.data ?? NO_AGENTS;
  const holding = useToolHolding();
  const [menu, setMenu] = useState(false);

  // Changing who is compared changes what the table's filters can name, so the table
  // starts over rather than keep a filter on an agent now gone.
  const setPicked = useCallback(
    (next: string[]) => {
      void setParams({
        ...CLEARED_TABLE_STATE,
        [SAFETY_AGENTS_PARAM]: next.length > 0 ? next : null,
      });
    },
    [setParams],
  );
  const byId = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const chosen = picked.flatMap((id) => {
    const agent = byId.get(id);
    return agent ? [agent] : [];
  });

  return (
    <HoldingContext.Provider value={holding}>
      <div className="tb">
        <div className="pk">
          {chosen.map((agent) => (
            <span key={agent.id} className="pk-c">
              <Tile agent={agent} s={18} />
              {agent.name}
              <button
                type="button"
                aria-label={t("Remove {0} from the comparison", agent.name)}
                onClick={() => setPicked(picked.filter((id) => id !== agent.id))}
              >
                <Ic n="x" s={10} />
              </button>
            </span>
          ))}
          {picked.length < MAX_COMPARED_AGENTS && (
            <div className="rel">
              <button type="button" className="btn sm" onClick={() => setMenu((open) => !open)}>
                <Ic n="plus" s={12} />
                {picked.length > 0 ? t("Compare another") : t("Pick an agent")}
              </button>
              {menu && (
                <Menu
                  label={t("Pick an agent")}
                  onClose={() => setMenu(false)}
                  items={[
                    { kind: "heading", label: t("Up to {0} agents", MAX_COMPARED_AGENTS) },
                    ...agents
                      .filter((agent) => !picked.includes(agent.id))
                      .map((agent) => ({
                        kind: "item" as const,
                        icon: <Tile agent={agent} s={18} />,
                        label: agent.name,
                        note: t(
                          "{0, plural, one {# tool} other {# tools}} · ceiling {1}",
                          agent.toolNames.length,
                          tierLabel(t, agent.autonomyCeiling),
                        ),
                        onSelect: () => setPicked([...picked, agent.id]),
                      })),
                  ]}
                />
              )}
            </div>
          )}
        </div>
        <span className="sp" />
      </div>
      {picked.length === 0 ? (
        <div className="pk-e">
          <b>{t("Pick an agent to see what it can do without a person")}</b>
          <span>
            {t(
              "Each tool is answered twice: for a run that has read only your own records, and for one that has read an email, a document or another message written outside the organization.",
            )}
          </span>
          <div className="ms">
            {agents.map((agent) => (
              <button
                key={agent.id}
                type="button"
                className={cn("ms-i", !agent.enabled && "off")}
                title={agent.name}
                aria-label={agent.name}
                onClick={() => setPicked([agent.id])}
              >
                <Tile agent={agent} s={32} />
              </button>
            ))}
          </div>
        </div>
      ) : (
        <>
          <AgentHeaders agentIds={picked} />
          <AgentToolsTable agentIds={picked} />
        </>
      )}
    </HoldingContext.Provider>
  );
}

function useAgentHeaders(agentIds: readonly string[]) {
  return useQuery({
    ...queries.agentSafety.headers(agentIds),
    staleTime: AGENT_SAFETY_STALE_MS,
  });
}

function AgentHeaders({ agentIds }: { agentIds: readonly string[] }) {
  const t = useT();
  const query = useAgentHeaders(agentIds);
  const holding = useContext(HoldingContext);

  if (query.isError) {
    return (
      <div className="bnr d" role="alert">
        <Ic n="alert" s={14} />
        <span>{t("What these agents can do without a person could not be loaded.")}</span>
        <button type="button" className="btn sm" onClick={() => void query.refetch()}>
          {t("Try again")}
        </button>
      </div>
    );
  }

  const found = new Map((query.data ?? []).map((header) => [header.agentId, header]));

  return (
    <div className="ahs">
      {agentIds.map((agentId) => {
        const header = found.get(agentId);
        return header ? (
          <AgentHeaderRow
            key={agentId}
            header={header}
            agent={holding.agents.get(agentId) ?? null}
          />
        ) : null;
      })}
    </div>
  );
}

function AgentHeaderRow({
  header,
  agent,
}: {
  header: AgentSafetyHeader;
  agent: AgentDefinitionRow | null;
}) {
  const t = useT();
  const rt = useRichT();
  const navigate = useAIControlNavigation();
  const [listed, setListed] = useState(false);

  return (
    <div className="ah">
      <div className="ah-h">
        <Tile agent={agent ?? header.agent} s={28} />
        <div className="sh-t">
          <b>
            {header.agent.name}
            {!header.agent.enabled && <span className="tg">{t("Off")}</span>}
            {agent?.shadowMode && (
              <span className="tg">
                <Ic n="eyeOff" s={10} />
                {t("Shadow")}
              </span>
            )}
          </b>
          <span>
            {t(
              "{0} · ceiling {1}",
              reachLabel(t, header),
              tierLabel(t, header.agent.autonomyCeiling),
            )}
          </span>
        </div>
      </div>
      {header.reach.warnings.map((warning) =>
        warning.kind === "OpenWithSensitiveTools" ? (
          <div key={warning.kind} className="ah-w">
            <Ic n="warn" s={13} />
            <span>
              {rt(
                "Open to everyone, and holds <tools>{0, plural, one {# tool that leaves the organization} other {# tools that leave the organization}}</tools>.",
                {
                  tools: (children) => (
                    <button
                      type="button"
                      className="lnk"
                      onClick={() => setListed((open) => !open)}
                    >
                      {children}
                    </button>
                  ),
                },
                warning.tools.length,
              )}
              {listed && (
                <em>
                  {" "}
                  {warning.tools.map((name) => describeToolCall(name, null).title).join(", ")}.
                </em>
              )}
            </span>
            <button
              type="button"
              className="btn sm"
              onClick={() =>
                navigate({ tab: "agents", panel: { mode: "edit", entityId: header.agentId } })
              }
            >
              {t("Limit to roles")}
            </button>
          </div>
        ) : (
          <div key={warning.kind} className="ah-w">
            <Ic n="warn" s={13} />
            <span>
              {t(
                "This agent is restricted to roles and no role is granted it, so nobody can use it.",
              )}
            </span>
          </div>
        ),
      )}
    </div>
  );
}

function AgentToolsTable({ agentIds }: { agentIds: readonly string[] }) {
  const t = useT();
  const headers = useAgentHeaders(agentIds);
  const summary = useQuery({
    ...queries.agentSafety.summary(),
    staleTime: SAFETY_SUMMARY_STALE_MS,
  });
  const resources = summary.data?.resources ?? NO_RESOURCES;
  const known = headers.data ?? NO_HEADERS;

  const agents = useMemo(
    () => known.map((header) => ({ id: header.agentId, name: header.agent.name })),
    [known],
  );
  const graphql = useMemo(() => createAgentToolSafetyTableGraphQLConfig(agentIds), [agentIds]);
  const columns = useMemo(() => getAgentToolColumns(t, agents, resources), [agents, resources, t]);

  return (
    <DataTable<AgentToolSafetyRow>
      name="Agent Tool"
      emptyTitle={t("No agent tools yet")}
      queryKey={AGENT_TOOL_SAFETY_LIST_KEY}
      graphql={graphql}
      resource={Resource.AgentDefinition}
      columns={columns}
      TablePanel={AgentToolSheet}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ maxTier: false, needs: false }}
    />
  );
}

function AgentToolSheet({ open, onOpenChange, row }: DataTablePanelProps<AgentToolSafetyRow>) {
  const holding = useContext(HoldingContext);

  return (
    <ToolSheet
      policy={open && row ? row.policy : null}
      holders={row ? holdersOf(holding, row.policyName) : []}
      focus={row?.agentId ?? null}
      onClose={() => onOpenChange(false)}
    />
  );
}
