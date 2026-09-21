import { useT } from "@trenova/shared/i18n/use-t";
import { EmptyState } from "@/components/empty-state";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { apiService } from "@/services/api";
import type { PanelMode } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, BotIcon, PlusIcon, SearchIcon, WrenchIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { filterAgents, groupAgentsByTrigger } from "./agent-roster";
import { AgentShelves } from "./agent-rows";
import { toAgentPanelRow, toSaveRequest, type AgentPanelRow } from "./agent-form-schema";
import { AgentPanel } from "./agent-panel";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

type PanelState = { open: boolean; mode: PanelMode; row: AgentPanelRow | null };

export default function AgentsTab() {
  const t = useT();
  const queryClient = useQueryClient();

  const listQuery = useQuery(queries.assistant.agents(false));
  const templatesQuery = useQuery(queries.assistant.agentTemplates());

  const { allowed: canCreate } = usePermission(Resource.AgentDefinition, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.AgentDefinition, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.AgentDefinition, Operation.Delete);
  const { allowed: canRun } = usePermission(Resource.AgentRun, Operation.Create);

  const [panel, setPanel] = useState<PanelState>({ open: false, mode: "create", row: null });
  const [deleting, setDeleting] = useState<AgentDefinitionRow | null>(null);
  const [query, setQuery] = useState("");

  const agents = useMemo(() => listQuery.data ?? [], [listQuery.data]);
  const shelves = useMemo(() => groupAgentsByTrigger(filterAgents(agents, query)), [agents, query]);
  const templates = templatesQuery.data?.templates ?? [];

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.assistant._def });
  }, [queryClient]);

  const toggleMutation = useApiMutation({
    mutationFn: ({ agent, enabled }: { agent: AgentDefinitionRow; enabled: boolean }) =>
      apiService.agentDefinitionService.update(agent.id, {
        ...toSaveRequest(toAgentPanelRow(agent)),
        enabled,
      }),
    onSuccess: async (_result, { enabled }) => {
      toast.success(enabled ? t("Agent enabled") : t("Agent disabled"));
      await invalidate();
    },
    resourceName: "Agent",
  });

  const runMutation = useApiMutation({
    mutationFn: (agent: AgentDefinitionRow) =>
      apiService.agentRunService.start({ agentDefinitionId: agent.id }),
    onSuccess: async () => {
      toast.success(t("Run started"));
      await invalidate();
    },
    resourceName: "Agent Run",
  });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.agentDefinitionService.remove(id),
    onSuccess: async () => {
      toast.success(t("Agent removed"));
      setDeleting(null);
      await invalidate();
    },
    resourceName: "Agent",
  });

  const openCreate = useCallback(() => setPanel({ open: true, mode: "create", row: null }), []);
  const openEdit = useCallback(
    (agent: AgentDefinitionRow) =>
      setPanel({ open: true, mode: "edit", row: toAgentPanelRow(agent) }),
    [],
  );

  const enabledCount = agents.filter((agent) => agent.enabled).length;

  return (
    <section className="flex flex-col gap-4">
      {agents.length > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-2">
          <Input
            inputContainerClassName="w-full max-w-xs"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t("Search agents")}
            className="h-8"
            leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            aria-label={t("Search agents")}
          />
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground text-xs tabular-nums">
              {t("{0} of {1} on", enabledCount, agents.length)}
            </span>
            {canCreate && (
              <Button size="sm" onClick={openCreate}>
                <PlusIcon className="size-3.5" />
                {t("New agent")}
              </Button>
            )}
          </div>
        </div>
      )}

      {listQuery.isLoading ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-14" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <div className="flex justify-center py-6">
          <EmptyState
            icons={[BotIcon, AssistMark, WrenchIcon]}
            title={t("No agents yet")}
            description={t(
              "An agent is a set of instructions, a choice of tools and a trigger. Build one from a template in a minute, or write exactly the agent your operation needs.",
            )}
            action={
              canCreate ? { icon: PlusIcon, label: t("New agent"), onClick: openCreate } : undefined
            }
          />
        </div>
      ) : shelves.length === 0 ? (
        <p className="text-muted-foreground px-1 py-6 text-center text-sm">
          {t("No agents match that search.")}
        </p>
      ) : (
        <AgentShelves
          shelves={shelves}
          templates={templates}
          actions={{
            canUpdate,
            canDelete,
            canRun,
            isToggling: (agent) =>
              toggleMutation.isPending && toggleMutation.variables?.agent.id === agent.id,
            isRunning: (agent) => runMutation.isPending && runMutation.variables?.id === agent.id,
            onEdit: openEdit,
            onToggleEnabled: (agent, enabled) => toggleMutation.mutate({ agent, enabled }),
            onRunNow: (agent) => runMutation.mutate(agent),
            onDelete: setDeleting,
          }}
        />
      )}

      <AgentPanel
        open={panel.open}
        onOpenChange={(open) => setPanel((current) => ({ ...current, open }))}
        mode={panel.mode}
        row={panel.row}
      />

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <AlertTriangleIcon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Remove agent")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Remove {0}. Existing conversations with this agent will remain but cannot be continued, and its schedule stops.",
                deleting?.name ?? t("this agent"),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              disabled={deleteMutation.isPending}
            >
              {t("Remove agent")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </section>
  );
}
