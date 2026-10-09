import { DataTable } from "@/components/data-table/data-table";
import { usePermission } from "@/hooks/use-permission";
import { agentControlQueryOptions } from "@/lib/graphql/agent-control";
import {
  AGENT_MEMORY_LIST_KEY,
  agentMemoryTableGraphQLConfig,
  agentMemoryTotalQueryKey,
  fetchAgentMemoryTotal,
  setAgentMemoryStatus,
  type AgentMemoryRow,
} from "@/lib/graphql/agent-memories";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArchiveIcon, ArchiveRestoreIcon } from "@trenova/shared/components/icons";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryStates } from "nuqs";
import { useMemo } from "react";
import { toast } from "sonner";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import {
  MEMORY_STARTER_PARAM,
  memoryStarterKinds,
  memoryStarterParser,
  type MemoryStarterKind,
} from "../../ai-control-tabs";
import { useAIControlNavigation } from "../../use-ai-control-navigation";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import { AgentReflections } from "./agent-reflections";
import { getMemoryColumns } from "./memory-columns";
import type { MemoryFormValues } from "./memory-form-schema";
import { MemoryKindMark } from "./memory-kind";
import { MemoryPanel } from "./memory-panel";
import { MemorySuggestions } from "./memory-suggestions";
import { MemoryUsageNotice } from "./memory-usage-notice";

const starterParsers = {
  ...panelSearchParamsParser,
  [MEMORY_STARTER_PARAM]: memoryStarterParser,
};

const EXAMPLES: Record<MemoryStarterKind, string> = defineLabels({
  Instruction: "Always CC the customer's AP inbox on invoices for …",
  Fact: "The receiving dock at … closes at …",
  Procedure: "To clear a rate mismatch: …",
});

/**
 * What the organization has told its agents. Every row here is read into the prompt of
 * every agent that asks for memory, so the list is also the place to see what an agent
 * recorded on its own, approve what one suggests, and retire what no longer holds.
 * Retiring keeps the row: what an agent was told last month is still worth reading.
 */
export default function MemoryTab() {
  const t = useT();
  const rt = useRichT();
  const queryClient = useQueryClient();
  const go = useAIControlNavigation();
  const columns = useMemo(() => getMemoryColumns(t), [t]);
  const { allowed: canUpdate } = usePermission(Resource.AgentMemory, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.AgentMemory, Operation.Create);
  const totalQuery = useQuery({
    queryKey: agentMemoryTotalQueryKey,
    queryFn: ({ signal }) => fetchAgentMemoryTotal({ signal }),
  });
  const retrievalQuery = useQuery(queries.aiRetrieval.status());
  const controlQuery = useQuery(agentControlQueryOptions());
  const [{ panelType, [MEMORY_STARTER_PARAM]: starterKind }, setStarter] =
    useQueryStates(starterParsers);

  const byMeaning = Boolean(retrievalQuery.data?.settings.activeModelKey);
  const learningOff = controlQuery.data?.learningOff ?? false;
  const empty = totalQuery.data === 0;
  // With nothing to list there is no table to hold the create panel, so a link
  // asking for one, or for a first memory of a kind, opens it here instead.
  const starting = useMemo<Partial<MemoryFormValues> | null>(
    () =>
      starterKind
        ? { kind: starterKind, content: t(EXAMPLES[starterKind]).replace(/ ?…$/, " ") }
        : null,
    [starterKind, t],
  );
  const starterOpen = canCreate && empty && (starterKind !== null || panelType === "create");

  const setStatus = async (row: Row<AgentMemoryRow>, status: AgentMemoryRow["status"]) => {
    await setAgentMemoryStatus(row.original.id, status);
    toast.success(status === "Retired" ? t("Memory retired") : t("Memory restored"));
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
      invalidateAIControlCounts(queryClient),
    ]);
  };

  const contextMenuActions: RowAction<AgentMemoryRow>[] = [
    {
      id: "retire",
      label: t("Retire"),
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => void setStatus(row, "Retired"),
      hidden: (row) => !canUpdate || row.original.status !== "Active",
    },
    {
      id: "restore",
      label: t("Restore"),
      icon: ArchiveRestoreIcon,
      onClick: (row) => void setStatus(row, "Active"),
      hidden: (row) => !canUpdate || row.original.status !== "Retired",
    },
  ];

  return (
    <div className="tabp">
      <p className="lead">
        {t("Agents read these before they answer.")}
        {retrievalQuery.data && !byMeaning && (
          <>
            {" "}
            {rt("They're found by their words until <link>search by meaning</link> is on.", {
              link: (children) => (
                <button type="button" className="lnk" onClick={() => go({ tab: "retrieval" })}>
                  {children}
                </button>
              ),
            })}
          </>
        )}
        {learningOff && <> {t("Learning is off, so agents won't suggest new ones.")}</>}
      </p>
      <MemoryUsageNotice />
      <MemorySuggestions canDecide={canUpdate} />
      <AgentReflections />
      <section className="sec">
        {empty ? (
          <div className="mem-e">
            <b>{t("Nothing recorded yet")}</b>
            <span>
              {t(
                "Tell agents something once and every one of them remembers it — a customer's rule, a dock's hours, how your team clears a hold.",
              )}
              {!learningOff &&
                ` ${t("With Learn from their work on, they'll also suggest lessons for you to approve.")}`}
            </span>
            {canCreate && (
              <div className="mem-x">
                {memoryStarterKinds.map((kind) => (
                  <button
                    key={kind}
                    type="button"
                    onClick={() => void setStarter({ [MEMORY_STARTER_PARAM]: kind })}
                  >
                    <MemoryKindMark kind={kind} />
                    {t(EXAMPLES[kind])}
                  </button>
                ))}
              </div>
            )}
          </div>
        ) : (
          <DataTable<AgentMemoryRow>
            name="Memory"
            emptyTitle={t("No memories match")}
            queryKey={AGENT_MEMORY_LIST_KEY}
            graphql={agentMemoryTableGraphQLConfig}
            resource={Resource.AgentMemory}
            columns={columns}
            contextMenuActions={contextMenuActions}
            TablePanel={MemoryPanel}
            initialColumnVisibility={{
              source: false,
              toolName: false,
              createdAt: false,
              expiresAt: false,
              lastUsedAt: false,
            }}
          />
        )}
      </section>
      <MemoryPanel
        open={starterOpen}
        mode="create"
        row={null}
        preset={starting}
        onOpenChange={(open) =>
          !open &&
          void setStarter({ panelType: null, panelEntityId: null, [MEMORY_STARTER_PARAM]: null })
        }
      />
    </div>
  );
}
