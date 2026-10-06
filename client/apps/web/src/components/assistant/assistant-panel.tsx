import { DeskThread } from "@/components/desk-chat/desk-thread";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AssistantLayout } from "@/lib/assistant-dock";
import { conversationPath } from "@/lib/conversation-path";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AssistantThread } from "@/types/assistant";
import { Trash01Icon } from "@trenova/shared/components/icons";
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
import { useT } from "@trenova/shared/i18n/use-t";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  AssistantFullHeader,
  AssistantHeader,
  type AssistantThreadActions,
  type AssistantView,
} from "./assistant-header";
import { AssistantHistory } from "./assistant-history";
import { AssistantHome } from "./assistant-home";
import { AssistantSidebar } from "./assistant-sidebar";
import type { AssistantThreads } from "./use-assistant-threads";
import { useStartConversation } from "./use-start-conversation";

type AssistantPanelProps = {
  layout: AssistantLayout;
  view: AssistantView;
  data: AssistantThreads;
  onHistory: (open: boolean) => void;
  onLayout: (layout: AssistantLayout) => void;
  onClose: () => void;
};

/**
 * The panel's contents. In the corner and docked to the side: a header, then
 * the home, the conversations, or the open conversation. Full screen: the
 * conversations down the side, and the open conversation or the front page
 * beside them. The conversation itself is the Desk's thread, compact in the
 * narrow layouts and at the Desk's own sizes full screen, with what only the
 * Desk has (its workspace, schedules, chapters) left to the Desk: "Open in
 * Desk" carries the conversation there.
 */
export function AssistantPanel({
  layout,
  view,
  data,
  onHistory,
  onLayout,
  onClose,
}: AssistantPanelProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const sidebarCollapsed = useAssistantStore((state) => state.sidebarCollapsed);
  const setSidebarCollapsed = useAssistantStore((state) => state.setSidebarCollapsed);
  const toggleSidebar = useAssistantStore((state) => state.toggleSidebar);
  const [deleting, setDeleting] = useState<AssistantThread | null>(null);
  const [searchSignal, setSearchSignal] = useState(0);
  const { threads, agents, agentsById, activeThread, agentsUnavailable, isLoading } = data;
  const full = layout === "full";
  const activeAgent = activeThread
    ? (agentsById.get(activeThread.agentDefinitionId) ?? null)
    : null;

  const open = useCallback(
    (thread: AssistantThread) => {
      setActiveThreadId(thread.id);
      onHistory(false);
    },
    [onHistory, setActiveThreadId],
  );
  const goHome = useCallback(() => {
    setActiveThreadId(null);
    onHistory(false);
  }, [onHistory, setActiveThreadId]);
  const { start, isStarting } = useStartConversation({ origin: "Panel", onStarted: open });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.deleteThread(id),
    onSuccess: async (_result, id) => {
      toast.success(t("Conversation deleted"));
      setDeleting(null);
      if (activeThread?.id === id) {
        setActiveThreadId(null);
      }
      await queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });
    },
    resourceName: "Conversation",
  });

  const openInDesk = useCallback(
    (threadId: string, artifactId?: string | null) => {
      onClose();
      const path = conversationPath(threadId);
      void navigate(artifactId ? `${path}?a=${encodeURIComponent(artifactId)}` : path);
    },
    [navigate, onClose],
  );
  const actions = useMemo<AssistantThreadActions>(
    () => ({
      onDelete: setDeleting,
      onDownloadTranscript: (thread) => downloadAssistantTranscript(thread.id),
      onOpenInDesk: (thread) => openInDesk(thread.id),
    }),
    [openInDesk],
  );

  // Full screen, ⌘K searches the conversations here rather than opening the
  // app's palette behind the panel, and ⌘\ folds the list away. Caught on the
  // way down so the palette's own binding never sees the key.
  useEffect(() => {
    if (!full) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (!(event.metaKey || event.ctrlKey) || event.altKey || event.shiftKey) {
        return;
      }
      const key = event.key.toLowerCase();
      if (key === "k") {
        event.preventDefault();
        event.stopImmediatePropagation();
        setSidebarCollapsed(false);
        setSearchSignal((value) => value + 1);
      } else if (key === "\\") {
        event.preventDefault();
        event.stopImmediatePropagation();
        toggleSidebar();
      }
    };
    window.addEventListener("keydown", onKeyDown, { capture: true });
    return () => window.removeEventListener("keydown", onKeyDown, { capture: true });
  }, [full, setSidebarCollapsed, toggleSidebar]);

  const thread =
    view === "thread" && activeThread ? (
      <DeskThread
        key={activeThread.id}
        thread={activeThread}
        agent={activeAgent}
        agentsUnavailable={agentsUnavailable}
        density={full ? "regular" : "compact"}
        threads={threads}
        onSwitchAgent={(agentId, draft) => start(agentId, draft)}
        onStartNew={activeAgent && !isStarting ? () => start(activeAgent.id) : undefined}
        pageSource="screen"
        onOpenInDesk={(artifactId) => openInDesk(activeThread.id, artifactId)}
        disclaimer={t(
          "The assistant can make mistakes. Check important details before you act on them.",
        )}
      />
    ) : null;
  const home = (
    <AssistantHome
      full={full}
      agents={agents}
      agentsById={agentsById}
      threads={threads}
      isLoading={isLoading}
      isStarting={isStarting}
      onStart={start}
      onOpen={open}
      onShowAll={() => onHistory(true)}
    />
  );

  return (
    <>
      {full ? (
        <>
          <AssistantSidebar
            threads={threads}
            agentsById={agentsById}
            activeThreadId={activeThread?.id ?? null}
            searchSignal={searchSignal}
            onOpen={open}
            onNew={goHome}
          />
          <div className="as-main">
            <AssistantFullHeader
              sidebarOpen={!sidebarCollapsed}
              thread={view === "thread" ? activeThread : null}
              actions={actions}
              onToggleSidebar={toggleSidebar}
              onNew={goHome}
              onShrink={() => onLayout("compact")}
              onClose={onClose}
            />
            {thread ?? home}
          </div>
        </>
      ) : (
        <div className="as-main">
          <AssistantHeader
            layout={layout}
            view={view}
            thread={activeThread}
            actions={actions}
            onBack={() => (view === "history" && activeThread ? onHistory(false) : goHome())}
            onToggleHistory={() => onHistory(view !== "history")}
            onNew={goHome}
            onLayout={onLayout}
            onClose={onClose}
          />
          <div className="as-body">
            {view === "history" ? (
              <AssistantHistory
                threads={threads}
                agentsById={agentsById}
                activeThreadId={activeThread?.id ?? null}
                onOpen={open}
                onNew={goHome}
              />
            ) : (
              (thread ?? home)
            )}
          </div>
        </div>
      )}

      <AlertDialog open={deleting !== null} onOpenChange={(next) => !next && setDeleting(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogMedia>
              <Trash01Icon />
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
