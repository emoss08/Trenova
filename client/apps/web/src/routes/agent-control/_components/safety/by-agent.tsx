import { DataTable } from "@/components/data-table/data-table";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { describeToolCall } from "@/components/assistant/tool-presentation";
import { searchParamsParser } from "@/hooks/data-table/use-data-table-state";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import {
  AGENT_TOOL_SAFETY_LIST_KEY,
  createAgentToolSafetyTableGraphQLConfig,
  type AgentSafetyHeader,
  type AgentToolSafetyRow,
} from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Resource } from "@trenova/shared/types/permission";
import { CircleAlertIcon, TriangleAlertIcon, XIcon } from "lucide-react";
import { useQueryStates } from "nuqs";
import { useCallback, useMemo } from "react";
import {
  CLEARED_TABLE_STATE,
  SAFETY_AGENTS_PARAM,
  safetyAgentsParser,
} from "../../ai-control-tabs";
import { AgentPicker, MAX_COMPARED_AGENTS } from "./agent-picker";
import { getAgentToolColumns } from "./safety-columns";
import { SAFETY_SUMMARY_STALE_MS } from "./safety-figures";
import { reachLabel, tierLabel } from "./safety-model";
import { AgentToolPanel } from "./safety-panels";

/** An agent's answers move with its settings and trust, so they are re-read after a minute. */
const AGENT_SAFETY_STALE_MS = 60_000;

const NO_RESOURCES: readonly string[] = [];
const NO_HEADERS: readonly AgentSafetyHeader[] = [];

const byAgentParsers = {
  ...searchParamsParser,
  [SAFETY_AGENTS_PARAM]: safetyAgentsParser,
};

/**
 * What the agents someone picks can do without a person, tool by tool: once
 * for a run that has read nothing from outside, once for a run that has.
 * Every picked agent's tools are one table, so they page, filter, sort and
 * scroll together; nothing is read until an agent is picked.
 */
export default function ByAgentView() {
  const t = useT();
  const [params, setParams] = useQueryStates(byAgentParsers);
  const stored = params[SAFETY_AGENTS_PARAM];
  const picked = useMemo(() => [...new Set(stored)].slice(0, MAX_COMPARED_AGENTS), [stored]);

  // Changing who is compared changes what the table's filters can name, so
  // the table starts over rather than keep a filter on an agent now gone.
  const setPicked = useCallback(
    (next: string[]) => {
      void setParams({
        ...CLEARED_TABLE_STATE,
        [SAFETY_AGENTS_PARAM]: next.length > 0 ? next : null,
      });
    },
    [setParams],
  );
  const add = useCallback(
    (agent: AgentChoice) => {
      if (!picked.includes(agent.id)) {
        setPicked([...picked, agent.id]);
      }
    },
    [picked, setPicked],
  );
  const remove = useCallback(
    (agentId: string) => setPicked(picked.filter((id) => id !== agentId)),
    [picked, setPicked],
  );

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <SectionPanel
        title={t("By agent")}
        count={picked.length > 0 ? picked.length : undefined}
        help={t(
          "Before outside text is a run that has read only the organization's own records. After outside text is a run that has read an email, a document or another message written outside the organization: anything it would send out waits for approval.",
        )}
        action={<AgentPicker picked={picked} onAdd={add} />}
      >
        {picked.length === 0 ? (
          <SectionPanelQuiet>
            {t("Add an agent to see what it can do without a person.")}
          </SectionPanelQuiet>
        ) : (
          <AgentHeaders agentIds={picked} onRemove={remove} />
        )}
      </SectionPanel>
      {picked.length > 0 ? <AgentToolsTable agentIds={picked} /> : null}
    </div>
  );
}

function useAgentHeaders(agentIds: readonly string[]) {
  return useQuery({
    ...queries.agentSafety.headers(agentIds),
    staleTime: AGENT_SAFETY_STALE_MS,
  });
}

function AgentHeaders({
  agentIds,
  onRemove,
}: {
  agentIds: readonly string[];
  onRemove: (agentId: string) => void;
}) {
  const t = useT();
  const query = useAgentHeaders(agentIds);

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-2 p-3" aria-busy>
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-3.5 w-72" />
      </div>
    );
  }

  if (query.isError) {
    return (
      <div className="p-3">
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t("What these agents can do without a person could not be loaded.")}
          </AlertDescription>
          <AlertAction>
            <Button variant="outline" size="xs" onClick={() => void query.refetch()}>
              {t("Try again")}
            </Button>
          </AlertAction>
        </Alert>
      </div>
    );
  }

  const found = new Map(query.data.map((header) => [header.agentId, header]));

  return (
    <div className="divide-border flex flex-col divide-y">
      {agentIds.map((agentId) => {
        const header = found.get(agentId);
        return header ? (
          <AgentHeader key={agentId} header={header} onRemove={onRemove} />
        ) : (
          <div key={agentId} className="flex items-center justify-between gap-2 px-3 py-3">
            <p className="text-muted-foreground text-xs">{t("This agent no longer exists.")}</p>
            <Button variant="ghost" size="xs" onClick={() => onRemove(agentId)}>
              {t("Remove")}
            </Button>
          </div>
        );
      })}
    </div>
  );
}

function AgentHeader({
  header,
  onRemove,
}: {
  header: AgentSafetyHeader;
  onRemove: (agentId: string) => void;
}) {
  const t = useT();

  return (
    <section aria-label={header.agent.name} className="flex flex-col gap-2 px-3 py-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <h4 className="truncate text-sm font-semibold">{header.agent.name}</h4>
          {!header.agent.enabled ? (
            <Badge variant="neutral" appearance="outline">
              {t("Off")}
            </Badge>
          ) : null}
        </div>
        <Button
          variant="ghost"
          size="icon-xs"
          aria-label={t("Remove {0} from the comparison", header.agent.name)}
          onClick={() => onRemove(header.agentId)}
        >
          <XIcon className="size-3.5" />
        </Button>
      </div>
      <DescriptionList layout="inline">
        <DescriptionItem label={t("Who can use it")}>{reachLabel(t, header)}</DescriptionItem>
        <DescriptionItem label={t("Ceiling")}>
          {tierLabel(t, header.agent.autonomyCeiling)}
        </DescriptionItem>
      </DescriptionList>
      {header.reach.warnings.map((warning) => (
        <Alert key={warning.kind} variant="warning" size="sm">
          <TriangleAlertIcon />
          <AlertDescription>
            {warning.kind === "OpenWithSensitiveTools"
              ? t(
                  "Everyone who can use the assistant can use this agent, and it holds tools that reach restricted data or leave the organization: {0}.",
                  warning.tools.map((name) => describeToolCall(name, null).title).join(", "),
                )
              : t(
                  "This agent is restricted to roles and no role is granted it, so nobody can use it.",
                )}
          </AlertDescription>
        </Alert>
      ))}
    </section>
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
      queryKey={AGENT_TOOL_SAFETY_LIST_KEY}
      graphql={graphql}
      resource={Resource.AgentDefinition}
      columns={columns}
      TablePanel={AgentToolPanel}
      enableCreateAction={false}
      enableReadOnlyPanel
      initialColumnVisibility={{ maxTier: false, needs: false }}
    />
  );
}
