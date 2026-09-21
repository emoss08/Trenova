import { AGENT_ACCENTS, resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useApiMutation } from "@/hooks/use-api-mutation";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
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
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  ChevronsUpDownIcon,
  DownloadIcon,
  PanelRightIcon,
  PinIcon,
  Trash2Icon,
} from "lucide-react";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Outlet, useNavigate } from "react-router";
import { toast } from "sonner";
import { ArtifactsPane } from "./artifacts/artifacts-pane";
import { DeskDirectory } from "./desk-directory";
import { DeskColumns, DeskShell, WorkingDot } from "./desk-shell";
import { DeskTitleField } from "./desk-title-field";
import { DeskWorkspaceEmpty } from "./desk-workspace-empty";

export type DeskContextValue = {
  threads: AssistantThread[];
  agents: AgentDefinitionRow[];
  agentsById: Map<string, AgentDefinitionRow>;
  agentsUnavailable: boolean;
  isLoading: boolean;
  isStarting: boolean;
  /** Opens a conversation, optionally with the question that prompted it. */
  start: (agentId: string, question?: string) => void;
  remove: (thread: AssistantThread) => void;
  togglePin: (thread: AssistantThread) => void;
  /** Told while a turn is running, so the room can light up for it. */
  setWorking: (working: boolean) => void;
  /** Told each artifact a streaming turn announces, so the workspace opens on it. */
  noteLiveArtifact: (artifactId: string) => void;
  /** Opens an artifact the transcript referred to. */
  openArtifact: (threadId: string, artifactId: string) => void;
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
 * The Desk itself: one room, the whole window, a conversation on the left
 * and what it produced on the right.
 *
 * The frame owns everything that outlives a single page inside it — the
 * conversations, the agents, the delete confirmation, and the state of the
 * workspace — because all three pages of the Desk share them and none of
 * them should reload when you move between them.
 *
 * It also owns the header, which means the header can say what the open
 * conversation is without the conversation having to draw a title bar of
 * its own. There is one strip at the top of the room, not one per column.
 */
export function DeskLayout({ activeThreadId }: { activeThreadId: string | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const setLastAgentId = useAssistantStore((state) => state.setLastAgentId);
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const togglePane = useDeskStore((state) => state.togglePane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);

  const [deleting, setDeleting] = useState<AssistantThread | null>(null);
  const [working, setWorking] = useState(false);
  const [liveArtifactIds, setLiveArtifactIds] = useState<string[]>([]);

  const threadsQuery = useQuery(queries.assistant.threads());
  const agentsQuery = useQuery(queries.assistant.agents(true, true));
  const threads = useMemo(() => threadsQuery.data?.items ?? [], [threadsQuery.data?.items]);
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const activeThread = useMemo(
    () => threads.find((thread) => thread.id === activeThreadId) ?? null,
    [activeThreadId, threads],
  );
  const activeAgent = activeThread
    ? (agentsById.get(activeThread.agentDefinitionId) ?? null)
    : null;
  const accent = activeAgent ? AGENT_ACCENTS[resolveAgentIdentity(activeAgent).accent] : undefined;

  // Leaving a conversation leaves its turn behind with it: a light still on
  // for work that finished in a thread you are no longer looking at is a lie
  // about the room. Adjusted during render rather than in an effect, so the
  // first frame of a new conversation is already dark.
  const [seenThreadId, setSeenThreadId] = useState(activeThreadId);
  if (seenThreadId !== activeThreadId) {
    setSeenThreadId(activeThreadId);
    setWorking(false);
    setLiveArtifactIds([]);
  }

  const refreshThreads = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
    [queryClient],
  );

  const startMutation = useApiMutation({
    mutationFn: ({ agentId }: { agentId: string; question?: string }) =>
      apiService.assistantService.startThread(agentId, { origin: "Desk" }),
    onSuccess: async (thread, { agentId, question }) => {
      setLastAgentId(agentId);
      // Handed over rather than sent here: the conversation is the only
      // place that knows the thread is empty and that history has loaded,
      // which is what keeps a reload from asking the same question twice.
      if (question !== undefined && question !== "") {
        setOpeningQuestion({ threadId: thread.id, text: question });
      }
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

  const renameMutation = useApiMutation({
    mutationFn: ({ id, title }: { id: string; title: string }) =>
      apiService.assistantService.updateThread(id, { title }),
    onSuccess: refreshThreads,
    resourceName: "Conversation",
  });

  // A turn that produces something opens the workspace on it, even if it was
  // folded away: the person asked for the thing it holds.
  const noteLiveArtifact = useCallback(
    (artifactId: string) => {
      setLiveArtifactIds((ids) => (ids.includes(artifactId) ? ids : [...ids, artifactId]));
      setPane("open");
    },
    [setPane],
  );

  const openArtifact = useCallback(
    (threadId: string, artifactId: string) => {
      setActiveArtifact(threadId, artifactId);
      setPane("open");
    },
    [setActiveArtifact, setPane],
  );

  const value = useMemo<DeskContextValue>(
    () => ({
      threads,
      agents,
      agentsById,
      agentsUnavailable: agentsQuery.isError,
      isLoading: threadsQuery.isLoading || agentsQuery.isLoading,
      isStarting: startMutation.isPending,
      start: (agentId, question) => startMutation.mutate({ agentId, question }),
      remove: setDeleting,
      togglePin: (thread) => pinMutation.mutate(thread),
      setWorking,
      noteLiveArtifact,
      openArtifact,
    }),
    [
      agents,
      agentsById,
      agentsQuery.isError,
      agentsQuery.isLoading,
      noteLiveArtifact,
      openArtifact,
      pinMutation,
      startMutation,
      threads,
      threadsQuery.isLoading,
    ],
  );

  const workspaceOpen = activeThread !== null && pane === "open";

  // The one shortcut the room has. A person reading a wide table wants the
  // conversation out of the way and then wants it back a sentence later, and
  // reaching for a button in the corner each time is the sort of friction
  // that makes a workspace feel like a web page.
  useEffect(() => {
    if (activeThread === null) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "\\" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        togglePane();
      }
    };
    window.addEventListener("keydown", onKeyDown);

    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeThread, togglePane]);

  return (
    <DeskContext.Provider value={value}>
      <DeskShell
        accent={accent}
        working={working}
        lead={
          <div className="flex shrink-0 items-center gap-1.5">
            <AgentTile agent={activeAgent} size="md" />
            <WorkingDot working={working} />
          </div>
        }
        title={
          activeThread ? (
            <DeskTitleField
              key={activeThread.id}
              title={activeThread.title}
              placeholder={
                activeAgent
                  ? t("Conversation with {0}", activeAgent.name)
                  : t("Untitled conversation")
              }
              onCommit={(title) => renameMutation.mutate({ id: activeThread.id, title })}
            />
          ) : (
            <span className="truncate px-2 text-sm font-medium">{t("Desk")}</span>
          )
        }
        actions={
          <>
            <DeskSwitcher
              threads={threads}
              agents={agents}
              activeThreadId={activeThreadId}
              isLoading={threadsQuery.isLoading || agentsQuery.isLoading}
              isStarting={startMutation.isPending}
              onStart={(agentId) => startMutation.mutate({ agentId })}
              onDelete={setDeleting}
              onTogglePin={(thread) => pinMutation.mutate(thread)}
            />

            {activeThread && (
              <>
                <span aria-hidden className="bg-desk-hairline mx-1 h-5 w-px" />
                <HeaderAction
                  label={activeThread.pinned ? t("Unpin conversation") : t("Pin conversation")}
                  pressed={activeThread.pinned}
                  onClick={() => pinMutation.mutate(activeThread)}
                >
                  <PinIcon className="size-4" />
                </HeaderAction>
                <HeaderAction
                  label={t("Download transcript")}
                  onClick={() => downloadAssistantTranscript(activeThread.id)}
                >
                  <DownloadIcon className="size-4" />
                </HeaderAction>
                <HeaderAction
                  label={t("Delete conversation")}
                  destructive
                  onClick={() => setDeleting(activeThread)}
                >
                  <Trash2Icon className="size-4" />
                </HeaderAction>
                <HeaderAction
                  label={pane === "open" ? t("Hide the workspace") : t("Show the workspace")}
                  pressed={pane === "open"}
                  hint="⌘\"
                  onClick={togglePane}
                >
                  <PanelRightIcon className="size-4" />
                </HeaderAction>
              </>
            )}
          </>
        }
      >
        <DeskColumns
          workspaceOpen={workspaceOpen}
          conversation={<Outlet />}
          workspace={
            activeThread ? (
              <ArtifactsPane
                key={activeThread.id}
                threadId={activeThread.id}
                liveArtifactIds={liveArtifactIds}
                onClose={() => setPane("closed")}
                className="h-full"
              />
            ) : (
              <DeskWorkspaceEmpty />
            )
          }
        />
      </DeskShell>

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

/** The directory, behind the one control that opens it. */
function DeskSwitcher(props: Omit<Parameters<typeof DeskDirectory>[0], "onNavigate">) {
  const t = useT();
  const [open, setOpen] = useState(false);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground hover:text-foreground h-8 gap-1.5 px-2"
          >
            <span className="text-xs">{t("Conversations")}</span>
            <ChevronsUpDownIcon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent align="end" sideOffset={8} className="ui-lift-float w-88 p-0">
        <DeskDirectory {...props} onNavigate={() => setOpen(false)} />
      </PopoverContent>
    </Popover>
  );
}

function HeaderAction({
  label,
  hint,
  pressed,
  destructive = false,
  onClick,
  children,
}: {
  label: string;
  /** The keystroke that does the same thing, shown in the tooltip. */
  hint?: string;
  pressed?: boolean;
  destructive?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={label}
            aria-pressed={pressed}
            className={cn(
              pressed ? "text-foreground" : "text-muted-foreground hover:text-foreground",
              destructive && "hover:text-destructive",
            )}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="bottom" className="flex items-center gap-2">
        {label}
        {hint && <kbd className="text-2xs opacity-70">{hint}</kbd>}
      </TooltipContent>
    </Tooltip>
  );
}
