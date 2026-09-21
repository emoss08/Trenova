import { useT } from "@trenova/shared/i18n/use-t";
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
import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { Trash2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { AssistantHeader } from "./assistant-header";
import { AssistantHome } from "./assistant-home";
import { MessageThread } from "./message-thread";
import { ThreadSidebar } from "./thread-sidebar";

type AssistantPanelProps = {
  expanded: boolean;
  onToggleExpanded: () => void;
  onClose: () => void;
};

/**
 * The panel's contents: header, the conversation or the launch pad, and in
 * expanded mode the full list of conversations down the side.
 */
export function AssistantPanel({ expanded, onToggleExpanded, onClose }: AssistantPanelProps) {
  const t = useT();
  const queryClient = useQueryClient();

  const activeThreadId = useAssistantStore((state) => state.activeThreadId);
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const [deleting, setDeleting] = useState<AssistantThread | null>(null);

  const threadsQuery = useQuery(queries.assistant.threads());
  const agentsQuery = useQuery(queries.assistant.agents(true, true));
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);

  const threads = useMemo(() => threadsQuery.data?.items ?? [], [threadsQuery.data?.items]);
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const activeThread = threads.find((thread) => thread.id === activeThreadId) ?? null;
  const activeAgent = activeThread
    ? (agentsById.get(activeThread.agentDefinitionId) ?? null)
    : null;

  const refreshThreads = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });
  }, [queryClient]);

  const startMutation = useApiMutation({
    mutationFn: (agentId: string) => apiService.assistantService.startThread(agentId),
    onSuccess: async (thread) => {
      setActiveThreadId(thread.id);
      await refreshThreads();
    },
    resourceName: "Conversation",
  });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.deleteThread(id),
    onSuccess: async (_result, id) => {
      toast.success(t("Conversation deleted"));
      setDeleting(null);
      if (activeThreadId === id) {
        setActiveThreadId(null);
      }
      await refreshThreads();
    },
    resourceName: "Conversation",
  });

  const isLoading = threadsQuery.isLoading || agentsQuery.isLoading;

  return (
    <>
      <AssistantHeader
        agents={agents}
        activeAgent={activeAgent}
        activeThread={activeThread}
        threads={threads}
        expanded={expanded}
        isStarting={startMutation.isPending}
        onStart={(agentId) => startMutation.mutate(agentId)}
        onSelectThread={setActiveThreadId}
        onDeleteThread={setDeleting}
        onToggleExpanded={onToggleExpanded}
        onClose={onClose}
      />

      <div className="flex min-h-0 flex-1">
        {expanded && (
          <ThreadSidebar
            threads={threads}
            agentsById={agentsById}
            activeThreadId={activeThreadId}
            isLoading={threadsQuery.isLoading}
            canStart={agents.length > 0 && !startMutation.isPending}
            onSelect={setActiveThreadId}
            onStart={() => {
              if (activeAgent) {
                startMutation.mutate(activeAgent.id);
              } else if (agents.length > 0) {
                startMutation.mutate(agents[0].id);
              }
            }}
            onDelete={setDeleting}
            className="hidden w-64 md:flex"
          />
        )}

        <section className="flex min-h-0 min-w-0 flex-1 flex-col">
          {activeThread ? (
            <MessageThread
              key={activeThread.id}
              thread={activeThread}
              agent={activeAgent}
              expanded={expanded}
              onStartNew={
                activeAgent && !startMutation.isPending
                  ? () => startMutation.mutate(activeAgent.id)
                  : undefined
              }
            />
          ) : (
            <AssistantHome
              agents={agents}
              threads={threads}
              isLoading={isLoading}
              isStarting={startMutation.isPending}
              canManageAgents={canManageAgents}
              onStart={(agentId) => startMutation.mutate(agentId)}
              onSelectThread={setActiveThreadId}
            />
          )}
        </section>
      </div>

      <AlertDialog open={deleting !== null} onOpenChange={(open) => !open && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <Trash2Icon />
            </AlertDialogMedia>
            <AlertDialogTitle>{t("Delete this conversation?")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "“{0}” and everything the assistant looked up in it will be removed. Proposals that were already approved are not undone.",
                deleting?.title || t("Untitled conversation"),
              )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Keep it")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => deleting && deleteMutation.mutate(deleting.id)}
              disabled={deleteMutation.isPending}
            >
              {t("Delete conversation")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
