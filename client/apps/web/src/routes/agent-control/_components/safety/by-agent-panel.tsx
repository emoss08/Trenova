import { SECTION_TABLE_PAGE_SIZE, SectionTable } from "@/components/data-table/section-table";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AgentSafety, AgentToolSafety } from "@/lib/graphql/agent-safety";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertAction, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { pageSlice } from "@trenova/shared/lib/utils";
import { CircleAlertIcon, TriangleAlertIcon, XIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { AgentPicker } from "./agent-picker";
import { PolicyDetails } from "./policy-details";
import { agentToolColumns } from "./safety-columns";
import { reachLabel, sortToolsByExposure, tierLabel } from "./safety-model";

/** An agent's answers move with its settings and trust, so they are re-read after a minute. */
const AGENT_SAFETY_STALE_MS = 60_000;

const toolRowId = (tool: AgentToolSafety) => tool.policyName;
const toolRowLabel = (tool: AgentToolSafety) => tool.policy.title;
const renderToolDetails = (tool: AgentToolSafety) => <PolicyDetails policy={tool.policy} />;

/**
 * What each picked agent can do without a person, tool by tool: once for a
 * run that has read nothing from outside, once for a run that has. Nothing
 * is read until an agent is picked, and each agent is read on its own, so
 * the panel costs what the person asked to see and no more.
 */
export function ByAgentPanel() {
  const t = useT();
  const [picked, setPicked] = useState<string[]>([]);

  const add = useCallback((agent: AgentChoice) => {
    setPicked((current) => (current.includes(agent.id) ? current : [...current, agent.id]));
  }, []);
  const remove = useCallback((agentId: string) => {
    setPicked((current) => current.filter((id) => id !== agentId));
  }, []);

  return (
    <SectionPanel
      title={t("By agent")}
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
        <div className="divide-border flex flex-col divide-y">
          {picked.map((agentId) => (
            <AgentMatrix key={agentId} agentId={agentId} onRemove={remove} />
          ))}
        </div>
      )}
    </SectionPanel>
  );
}

function AgentMatrix({
  agentId,
  onRemove,
}: {
  agentId: string;
  onRemove: (agentId: string) => void;
}) {
  const t = useT();
  const query = useQuery({
    ...queries.agentSafety.agent(agentId),
    staleTime: AGENT_SAFETY_STALE_MS,
  });
  const safety = query.data;

  if (query.isPending) {
    return (
      <div className="flex flex-col gap-2 p-3" aria-busy>
        <Skeleton className="h-4 w-48" />
        <Skeleton className="h-3.5 w-72" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  if (query.isError) {
    return (
      <div className="p-3">
        <Alert variant="destructive" size="sm">
          <CircleAlertIcon />
          <AlertDescription>
            {t("What this agent can do without a person could not be loaded.")}
          </AlertDescription>
          <AlertAction className="flex gap-1">
            <Button variant="outline" size="xs" onClick={() => void query.refetch()}>
              {t("Try again")}
            </Button>
            <Button variant="ghost" size="xs" onClick={() => onRemove(agentId)}>
              {t("Remove")}
            </Button>
          </AlertAction>
        </Alert>
      </div>
    );
  }

  if (!safety) {
    return (
      <div className="flex items-center justify-between gap-2 px-3 py-3">
        <p className="text-muted-foreground text-xs">{t("This agent no longer exists.")}</p>
        <Button variant="ghost" size="xs" onClick={() => onRemove(agentId)}>
          {t("Remove")}
        </Button>
      </div>
    );
  }

  return <AgentMatrixBody safety={safety} onRemove={onRemove} />;
}

function AgentMatrixBody({
  safety,
  onRemove,
}: {
  safety: AgentSafety;
  onRemove: (agentId: string) => void;
}) {
  const t = useT();
  const [pageIndex, setPageIndex] = useState(0);
  const [pageSize, setPageSize] = useState<number>(SECTION_TABLE_PAGE_SIZE);
  const columns = useMemo(() => agentToolColumns(t), [t]);
  const tools = useMemo(() => sortToolsByExposure(safety.tools), [safety.tools]);
  const titles = useMemo(
    () => new Map(safety.tools.map((tool) => [tool.policyName, tool.policy.title])),
    [safety.tools],
  );

  const lastPage = Math.max(0, Math.ceil(tools.length / pageSize) - 1);
  const currentPage = Math.min(pageIndex, lastPage);
  const rows = useMemo(
    () => pageSlice(tools, currentPage, pageSize),
    [currentPage, pageSize, tools],
  );

  const changePageSize = useCallback((size: number) => {
    setPageSize(size);
    setPageIndex(0);
  }, []);

  return (
    <section aria-label={safety.agent.name} className="flex flex-col">
      <div className="flex flex-col gap-3 px-3 py-3">
        <div className="flex items-center justify-between gap-2">
          <div className="flex min-w-0 flex-wrap items-center gap-2">
            <h4 className="truncate text-sm font-semibold">{safety.agent.name}</h4>
            {!safety.agent.enabled ? (
              <Badge variant="neutral" appearance="outline">
                {t("Off")}
              </Badge>
            ) : null}
          </div>
          <Button
            variant="ghost"
            size="icon-xs"
            aria-label={t("Remove {0} from the comparison", safety.agent.name)}
            onClick={() => onRemove(safety.agentId)}
          >
            <XIcon className="size-3.5" />
          </Button>
        </div>
        <DescriptionList layout="inline">
          <DescriptionItem label={t("Who can use it")}>{reachLabel(t, safety)}</DescriptionItem>
          <DescriptionItem label={t("Ceiling")}>
            {tierLabel(t, safety.agent.autonomyCeiling)}
          </DescriptionItem>
        </DescriptionList>
        {safety.reach.warnings.map((warning) => (
          <Alert key={warning.kind} variant="warning" size="sm">
            <TriangleAlertIcon />
            <AlertDescription>
              {warning.kind === "OpenWithSensitiveTools"
                ? t(
                    "Everyone who can use the assistant can use this agent, and it holds tools that reach restricted data or leave the organization: {0}.",
                    warning.tools.map((name) => titles.get(name) ?? name).join(", "),
                  )
                : t(
                    "This agent is restricted to roles and no role is granted it, so nobody can use it.",
                  )}
            </AlertDescription>
          </Alert>
        ))}
      </div>

      <div className="border-border border-t">
        <SectionTable
          label={t("Tools {0} holds", safety.agent.name)}
          columns={columns}
          rows={rows}
          getRowId={toolRowId}
          rowLabel={toolRowLabel}
          renderDetails={renderToolDetails}
          isLoading={false}
          empty={t("This agent holds no tools.")}
          pagination={{
            mode: "offset",
            pageIndex: currentPage,
            pageSize,
            totalCount: tools.length,
            onPageChange: setPageIndex,
            onPageSizeChange: changePageSize,
          }}
        />
      </div>
    </section>
  );
}
