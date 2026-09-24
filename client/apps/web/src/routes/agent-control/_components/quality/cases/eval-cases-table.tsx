import { useT } from "@trenova/shared/i18n/use-t";
import { DataTable } from "@/components/data-table/data-table";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  AGENT_EVAL_CASE_LIST_KEY,
  agentEvalCaseTableGraphQLConfig,
  replayAgentEvalCase,
  setAgentEvalCaseStatus,
  type AgentEvalCaseRow,
} from "@/lib/graphql/agent-eval-cases";
import { AGENT_EVALUATION_LIST_KEY } from "@/lib/graphql/agent-evaluations";
import type { AgentEvalCaseStatus } from "@trenova/graphql/generated/graphql";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  CircleCheckIcon,
  PauseCircleIcon,
  PlayIcon,
} from "lucide-react";
import { useMemo } from "react";
import { toast } from "sonner";
import { CASE_STATUS_MOVED } from "./eval-case-badges";
import { getEvalCaseColumns } from "./eval-case-columns";
import { nextStatuses } from "./eval-case-model";
import { EvalCasePanel } from "./eval-case-panel";

type StatusMove = { id: string; status: AgentEvalCaseStatus };

/**
 * The golden set: every question an agent is replayed against after a change.
 * Decided proposals arrive here as candidates on their own; an administrator
 * activates the ones worth keeping, writes cases by hand, and quarantines a
 * case that has stopped being fair.
 */
export function EvalCasesTable() {
  const t = useT();
  const queryClient = useQueryClient();
  const columns = useMemo(() => getEvalCaseColumns(t), [t]);
  const { allowed: canUpdate } = usePermission(Resource.AgentEvalSuite, Operation.Update);
  const { allowed: canCreate } = usePermission(Resource.AgentEvalSuite, Operation.Create);

  const move = useApiMutation<unknown, StatusMove>({
    mutationFn: ({ id, status }) => setAgentEvalCaseStatus(id, status),
    onSuccess: async (_data, { status }) => {
      toast.success(t(CASE_STATUS_MOVED[status]));
      await queryClient.invalidateQueries({ queryKey: [AGENT_EVAL_CASE_LIST_KEY] });
    },
    resourceName: t("Evaluation case"),
  });

  const replay = useApiMutation<unknown, string>({
    mutationFn: (id) => replayAgentEvalCase(id),
    onSuccess: async () => {
      toast.success(t("Replay started"), {
        description: t("The agent as it is now answers the case with every write simulated."),
      });
      await queryClient.invalidateQueries({ queryKey: [AGENT_EVALUATION_LIST_KEY] });
    },
    resourceName: t("Evaluation case"),
  });

  const movable = (row: Row<AgentEvalCaseRow>, status: AgentEvalCaseStatus) =>
    canUpdate && nextStatuses(row.original.status).includes(status);

  const contextMenuActions: RowAction<AgentEvalCaseRow>[] = [
    {
      id: "activate",
      label: t("Activate"),
      icon: CircleCheckIcon,
      onClick: (row) => move.mutate({ id: row.original.id, status: "Active" }),
      hidden: (row) => !movable(row, "Active") || row.original.status === "Retired",
    },
    {
      id: "restore",
      label: t("Restore"),
      icon: ArchiveRestoreIcon,
      onClick: (row) => move.mutate({ id: row.original.id, status: "Active" }),
      hidden: (row) => !movable(row, "Active") || row.original.status !== "Retired",
    },
    {
      id: "quarantine",
      label: t("Quarantine"),
      icon: PauseCircleIcon,
      onClick: (row) => move.mutate({ id: row.original.id, status: "Quarantined" }),
      hidden: (row) => !movable(row, "Quarantined"),
    },
    {
      id: "retire",
      label: t("Retire"),
      icon: ArchiveIcon,
      variant: "destructive",
      onClick: (row) => move.mutate({ id: row.original.id, status: "Retired" }),
      hidden: (row) => !movable(row, "Retired"),
    },
    {
      id: "replay",
      label: t("Replay now"),
      icon: PlayIcon,
      onClick: (row) => replay.mutate(row.original.id),
      hidden: (row) => !canCreate || row.original.status === "Retired",
    },
  ];

  return (
    <DataTable<AgentEvalCaseRow>
      name="Evaluation Case"
      queryKey={AGENT_EVAL_CASE_LIST_KEY}
      graphql={agentEvalCaseTableGraphQLConfig}
      resource={Resource.AgentEvalSuite}
      columns={columns}
      contextMenuActions={contextMenuActions}
      TablePanel={EvalCasePanel}
      enableCreateAction={canCreate}
      initialColumnVisibility={{ trigger: false, expiresAt: false, weight: false }}
    />
  );
}
