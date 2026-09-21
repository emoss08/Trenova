import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
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
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@trenova/shared/components/ui/resizable";
import { useT } from "@trenova/shared/i18n/use-t";
import { Trash2Icon } from "lucide-react";
import { createContext, useCallback, useContext, useMemo, useState } from "react";
import { Outlet, useNavigate } from "react-router";
import { toast } from "sonner";
import { DeskRail } from "./desk-rail";

export type DeskContextValue = {
  threads: AssistantThread[];
  agents: AgentDefinitionRow[];
  agentsById: Map<string, AgentDefinitionRow>;
  agentsUnavailable: boolean;
  isLoading: boolean;
  isStarting: boolean;
  start: (agentId: string) => void;
  remove: (thread: AssistantThread) => void;
  togglePin: (thread: AssistantThread) => void;
};

const DeskContext = createContext<DeskContextValue | null>(null);

export function useDesk(): DeskContextValue {
  const value = useContext(DeskContext);
  if (value === null) {
    throw new Error("useDesk must be used inside the Desk layout");
  }

  return value;
}

/**
 * The Desk's frame: the rail on the left, whichever page is open on the
 * right. Conversations and agents are read once here and handed down, so
 * the rail, the home page and an open conversation agree on what exists.
 */
export function DeskLayout({ activeThreadId }: { activeThreadId: string | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setLastAgentId = useDeskStore((state) => state.setLastAgentId);
  const [deleting, setDeleting] = useState<AssistantThread | null>(null);

  const threadsQuery = useQuery(queries.assistant.threads());
  const agentsQuery = useQuery(queries.assistant.agents(true, true));
  const threads = useMemo(() => threadsQuery.data?.items ?? [], [threadsQuery.data?.items]);
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const refreshThreads = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    [queryClient],
  );

  const startMutation = useApiMutation({
    mutationFn: (agentId: string) =>
      apiService.assistantService.startThread(agentId, { origin: "Desk" }),
    onSuccess: async (thread, agentId) => {
      setLastAgentId(agentId);
      await refreshThreads();
      void navigate(`/desk/t/${thread.id}`);
    },
    resourceName: "Conversation",
  });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.deleteThread(id),
    onSuccess: async (_result, id) => {
      toast.success(t("Conversation deleted"));
      setDeleting(null);
      await refreshThreads();
      if (activeThreadId === id) {
        void navigate("/desk");
      }
    },
    resourceName: "Conversation",
  });

  const pinMutation = useApiMutation({
    mutationFn: (thread: AssistantThread) =>
      apiService.assistantService.updateThread(thread.id, { pinned: !thread.pinned }),
    onSuccess: refreshThreads,
    resourceName: "Conversation",
  });

  const value = useMemo<DeskContextValue>(
    () => ({
      threads,
      agents,
      agentsById,
      agentsUnavailable: agentsQuery.isError,
      isLoading: threadsQuery.isLoading || agentsQuery.isLoading,
      isStarting: startMutation.isPending,
      start: (agentId) => startMutation.mutate(agentId),
      remove: setDeleting,
      togglePin: (thread) => pinMutation.mutate(thread),
    }),
    [agents, agentsById, agentsQuery.isError, agentsQuery.isLoading, pinMutation, startMutation, threads, threadsQuery.isLoading],
  );

  return (
    <DeskContext.Provider value={value}>
      <PageLayout
        fill
        className="p-0"
        pageHeaderProps={{
          title: t("Desk"),
          description: t("Your conversations with the agents, what they produced, and what they are asking you to decide."),
        }}
      >
        <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
          <ResizablePanel defaultSize="260px" minSize="220px" maxSize="360px" className="hidden md:block">
            <DeskRail
              threads={threads}
              agents={agents}
              activeThreadId={activeThreadId}
              isLoading={threadsQuery.isLoading}
              isStarting={startMutation.isPending}
              onStart={(agentId) => startMutation.mutate(agentId)}
              onDelete={setDeleting}
              onTogglePin={(thread) => pinMutation.mutate(thread)}
              className="h-full"
            />
          </ResizablePanel>
          <ResizableHandle />
          <ResizablePanel minSize="50%">
            <div className="bg-canvas flex h-full min-h-0 min-w-0 flex-col">
              <Outlet />
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </PageLayout>

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
    </DeskContext.Provider>
  );
}
