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
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { apiService } from "@/services/api";
import type { PanelMode } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, BotIcon, PlusIcon, SparklesIcon, WrenchIcon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { AgentCard } from "./agent-cards";
import { toAgentPanelRow, toSaveRequest, type AgentPanelRow } from "./agent-form-schema";
import { AgentPanel } from "./agent-panel";

type PanelState = { open: boolean; mode: PanelMode; row: AgentPanelRow | null };

export default function AgentsTab() {
  const t = useT();
  const queryClient = useQueryClient();

  const listQuery = useQuery(queries.assistant.agents(false));
  const templatesQuery = useQuery(queries.assistant.agentTemplates());

  const { allowed: canCreate } = usePermission(Resource.AgentDefinition, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.AgentDefinition, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.AgentDefinition, Operation.Delete);

  const [panel, setPanel] = useState<PanelState>({ open: false, mode: "create", row: null });
  const [deleting, setDeleting] = useState<AgentDefinitionRow | null>(null);

  const agents = listQuery.data ?? [];
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
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-base font-semibold">
            {t("Agents")}
            {agents.length > 0 && (
              <span className="bg-muted text-muted-foreground rounded-full px-2 py-0.5 text-[11px] font-medium tabular-nums">
                {t("{0} of {1} enabled", enabledCount, agents.length)}
              </span>
            )}
          </h2>
          <p className="text-muted-foreground max-w-prose text-sm">
            {t(
              "Each agent carries its own instructions, the tools it may call and how much it may do on its own. Start from a template or write one from scratch.",
            )}
          </p>
        </div>
        {canCreate && agents.length > 0 && (
          <Button size="sm" onClick={openCreate}>
            <PlusIcon className="size-3.5" />
            {t("Add agent")}
          </Button>
        )}
      </div>

      {listQuery.isLoading ? (
        <div className="grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-44" />
          ))}
        </div>
      ) : agents.length === 0 ? (
        <div className="flex justify-center py-6">
          <EmptyState
            icons={[BotIcon, SparklesIcon, WrenchIcon]}
            title={t("No agents yet")}
            description={t(
              "An agent is a set of instructions, a choice of tools and a trigger. Build one from a template in a minute, or write exactly the agent your operation needs.",
            )}
            action={
              canCreate
                ? { icon: PlusIcon, label: t("Build your first agent"), onClick: openCreate }
                : undefined
            }
          />
        </div>
      ) : (
        <div className="grid gap-3 md:grid-cols-2 2xl:grid-cols-3">
          {agents.map((agent) => (
            <AgentCard
              key={agent.id}
              agent={agent}
              templates={templates}
              canUpdate={canUpdate}
              canDelete={canDelete}
              isToggling={
                toggleMutation.isPending && toggleMutation.variables?.agent.id === agent.id
              }
              onEdit={() => openEdit(agent)}
              onToggleEnabled={(enabled) => toggleMutation.mutate({ agent, enabled })}
              onDelete={() => setDeleting(agent)}
            />
          ))}
        </div>
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
