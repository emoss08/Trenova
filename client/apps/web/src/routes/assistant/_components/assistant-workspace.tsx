import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent } from "@trenova/shared/components/ui/card";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BotIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useCallback, useState } from "react";
import { toast } from "sonner";
import { AgentPicker } from "./agent-picker";
import { MessageThread } from "./message-thread";

export function AssistantWorkspace() {
  const t = useT();
  const queryClient = useQueryClient();

  const [activeThreadId, setActiveThreadId] = useState<string | null>(null);
  const [pickerOpen, setPickerOpen] = useState(false);

  const threadsQuery = useQuery(queries.assistant.threads());
  // Only enabled agents can hold a conversation, so the picker asks for those.
  const agentsQuery = useQuery(queries.assistant.agents(true));

  const threads = threadsQuery.data?.items ?? [];
  const agents = agentsQuery.data?.results ?? [];

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
      <aside className="border-border flex w-72 shrink-0 flex-col border-r">
        <div className="border-border border-b p-3">
          <Button
            size="sm"
            className="w-full"
            onClick={() => setPickerOpen(true)}
            disabled={!hasAgents}
          >
            <PlusIcon className="size-4" />
            {t("New conversation")}
          </Button>
        </div>
        <ScrollArea className="flex-1">
          {threadsQuery.isLoading ? (
            <div className="space-y-2 p-3">
              <Skeleton className="h-12" />
              <Skeleton className="h-12" />
            </div>
          ) : (
            <ThreadList
              threads={threads}
              activeThreadId={activeThreadId}
              onSelect={setActiveThreadId}
              onDelete={(id) => deleteMutation.mutate(id)}
            />
          )}
        </ScrollArea>
      </aside>

      <section className="flex min-h-0 flex-1 flex-col">
        {!hasAgents && !agentsQuery.isLoading ? (
          <NoAgentsConfigured />
        ) : activeThread ? (
          <MessageThread key={activeThread.id} thread={activeThread} />
        ) : (
          <EmptyConversation onStart={() => setPickerOpen(true)} disabled={!hasAgents} />
        )}
      </section>

      <AgentPicker
        open={pickerOpen}
        onOpenChange={setPickerOpen}
        agents={agents}
        isPending={startMutation.isPending}
        onSelect={(agentId) => startMutation.mutate(agentId)}
      />
    </div>
  );
}

type ThreadListProps = {
  threads: AssistantThread[];
  activeThreadId: string | null;
  onSelect: (id: string) => void;
  onDelete: (id: string) => void;
};

function ThreadList({ threads, activeThreadId, onSelect, onDelete }: ThreadListProps) {
  const t = useT();

  if (threads.length === 0) {
    return <p className="text-muted-foreground p-3 text-sm">{t("No conversations yet.")}</p>;
  }

  return (
    <ul className="p-2">
      {threads.map((thread) => (
        <li key={thread.id}>
          <div
            className={`group flex items-center gap-1 rounded-md px-2 py-2 ${
              thread.id === activeThreadId ? "bg-muted" : "hover:bg-muted/60"
            }`}
          >
            <button
              type="button"
              className="min-w-0 flex-1 text-left"
              onClick={() => onSelect(thread.id)}
            >
              <span className="block truncate text-sm">
                {thread.title || t("Untitled conversation")}
              </span>
            </button>
            <button
              type="button"
              aria-label={t("Delete conversation")}
              className="text-muted-foreground hover:text-destructive opacity-0 transition-opacity group-hover:opacity-100"
              onClick={() => onDelete(thread.id)}
            >
              <Trash2Icon className="size-3.5" />
            </button>
          </div>
        </li>
      ))}
    </ul>
  );
}

function EmptyConversation({ onStart, disabled }: { onStart: () => void; disabled: boolean }) {
  const t = useT();

  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
          <BotIcon className="text-muted-foreground size-8" />
          <div>
            <p className="font-medium">{t("Start a conversation")}</p>
            <p className="text-muted-foreground text-sm">
              {t(
                "Ask about a shipment, a driver, or how to do something in Trenova. The assistant can look records up and propose changes for you to approve.",
              )}
            </p>
          </div>
          <Button size="sm" onClick={onStart} disabled={disabled}>
            {t("New conversation")}
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function NoAgentsConfigured() {
  const t = useT();

  return (
    <div className="flex flex-1 items-center justify-center p-6">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 py-10 text-center">
          <BotIcon className="text-muted-foreground size-8" />
          <div>
            <p className="font-medium">{t("No agents are available")}</p>
            <p className="text-muted-foreground text-sm">
              {t(
                "An administrator needs to configure and enable an agent, and connect an AI provider, before conversations can start.",
              )}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
