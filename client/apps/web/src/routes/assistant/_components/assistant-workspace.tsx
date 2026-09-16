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
import { Button } from "@trenova/shared/components/ui/button";
import { usePermission } from "@/hooks/use-permission";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BotIcon, PlugZapIcon, Trash2Icon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";
import { AgentPicker } from "./agent-picker";
import { AgentAvatar } from "./message-items";
import { MessageThread } from "./message-thread";
import { ThreadSidebar } from "./thread-sidebar";

export function AssistantWorkspace() {
  const t = useT();
  const queryClient = useQueryClient();

  const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);
  const [deleting, setDeleting] = useState<AssistantThread | null>(null);

  const threadsQuery = useQuery(queries.assistant.threads());
  // Only enabled agents can hold a conversation, so the picker asks for those.
  const agentsQuery = useQuery(queries.assistant.agents(true));
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);

  const threads = threadsQuery.data?.items ?? [];
  const agents = agentsQuery.data?.results ?? [];
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const refreshThreads = useCallback(async () => {
    await queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });
  }, [queryClient]);

  const startMutation = useApiMutation({
    mutationFn: (agentId: string) => apiService.assistantService.startThread(agentId),
    onSuccess: async (thread) => {
      setActiveThreadId(thread.id);
      setPickerOpen(false);
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

  // Not memoized: `threads` is a fresh array on every render, so a useMemo here
  // would recompute anyway while pretending not to. Scanning a sidebar-sized
  // list is cheaper than the illusion.
  const activeThread = threads.find((thread) => thread.id === activeThreadId) ?? null;
  const hasAgents = agents.length > 0;

  return (
    <div className="flex min-h-0 flex-1">
      <ThreadSidebar
        threads={threads}
        agentsById={agentsById}
        activeThreadId={activeThreadId}
        isLoading={threadsQuery.isLoading}
        canStart={hasAgents}
        onSelect={setActiveThreadId}
        onStart={() => setPickerOpen(true)}
        onDelete={setDeleting}
      />

      <section className="flex min-h-0 flex-1 flex-col">
        {!hasAgents && !agentsQuery.isLoading ? (
          <NoAgentsConfigured canManageAgents={canManageAgents} />
        ) : activeThread ? (
          <MessageThread
            key={activeThread.id}
            thread={activeThread}
            agent={agentsById.get(activeThread.agentDefinitionId) ?? null}
            onDelete={() => setDeleting(activeThread)}
          />
        ) : (
          <EmptyWorkspace onStart={() => setPickerOpen(true)} disabled={!hasAgents} />
        )}
      </section>

      <AgentPicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        agents={agents}
        isPending={startMutation.isPending}
        onSelect={(agentId) => startMutation.mutate(agentId)}
      />

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
    </div>
  );
}

function EmptyWorkspace({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
  const t = useT();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-5 p-6 text-center">
      <AgentAvatar className="size-14 rounded-2xl [&_svg]:size-7" />
      <div className="flex max-w-md flex-col gap-1.5">
        <p className="text-lg font-semibold">{t("Ask the assistant")}</p>
        <p className="text-muted-foreground text-sm">
          {t(
            "Where a shipment is, who is available, what is holding an invoice, how to do something in Trenova. It reads the records you can see and proposes changes for you to approve.",
          )}
        </p>
      </div>
      <Button onClick={onStart} disabled={disabled}>
        {t("New conversation")}
      </Button>
    </div>
  );
}

function NoAgentsConfigured({ canManageAgents }: { canManageAgents: boolean }) {
  const t = useT();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-5 p-6 text-center">
      <span className="bg-muted text-muted-foreground flex size-14 items-center justify-center rounded-2xl">
        <BotIcon className="size-7" />
      </span>
      <div className="flex max-w-md flex-col gap-1.5">
        <p className="text-lg font-semibold">{t("No agents are available")}</p>
        <p className="text-muted-foreground text-sm">
          {canManageAgents
            ? t(
                "Connect an AI provider and enable an agent in Agent Control, then come back here to start a conversation.",
              )
            : t(
                "An administrator needs to connect an AI provider and enable an agent before conversations can start.",
              )}
        </p>
      </div>
      {canManageAgents && (
        <Button variant="outline" nativeButton={false} render={<Link to="/admin/agent-control" />}>
          <PlugZapIcon className="size-4" />
          {t("Open Agent Control")}
        </Button>
      )}
    </div>
  );
}
