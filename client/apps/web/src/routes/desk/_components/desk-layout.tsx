import { AGENT_ACCENTS, resolveAgentIdentity } from "@/components/agent-identity/agent-identity";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useMediaQuery } from "@/hooks/use-media-query";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
import { railIsOpen, useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifactEvent, AssistantThread } from "@/types/assistant";
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
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Sheet, SheetContent, SheetTitle } from "@trenova/shared/components/ui/sheet";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { DownloadIcon, PanelRightIcon, PinIcon, Trash2Icon } from "lucide-react";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import { DeskWorkspace } from "./desk-workspace";
import { DESK_WIDE_QUERY } from "./desk-dimensions";
import { DeskRail, DeskRailStrip, RAIL_SHORTCUT } from "./desk-rail";
import { DeskColumns, DeskRailFold, DeskShell } from "./desk-shell";
import { DeskTitleField } from "./desk-title-field";
import { DeskWorkspaceEmpty } from "./desk-workspace-empty";

export type DeskContextValue = {
  threads: AssistantThread[];
  /**
   * The open conversation: from the list, or read on its own when the list
   * does not carry it — a palette question that was never kept, opened from
   * the notice that its answer is in.
   */
  activeThread: AssistantThread | null;
  agents: AgentChoice[];
  agentsById: Map<string, AgentChoice>;
  agentsUnavailable: boolean;
  isLoading: boolean;
  isStarting: boolean;
  /** Opens a conversation, optionally with the question that prompted it. */
  start: (agentId: string, question?: string) => void;
  remove: (thread: AssistantThread) => void;
  togglePin: (thread: AssistantThread) => void;
  /** Told while a turn is running, so the room can light up for it. */
  setWorking: (working: boolean) => void;
  /** Told what a streaming turn has produced so far, so the workspace opens on the newest and follows the set. */
  noteLiveArtifacts: (artifacts: readonly AssistantArtifactEvent[]) => void;
  /** Opens an artifact the transcript referred to. */
  openArtifact: (threadId: string, artifactId: string) => void;
};

const DeskContext = createContext<DeskContextValue | null>(null);

/** What a streaming turn has produced so far, and how many times that has changed. */
export type LiveArtifacts = {
  /** The artifacts the turn still holds, newest last. */
  ids: readonly string[];
  /** Bumped on every change to the set, so a reader re-reads it even when the newest id stays. */
  revision: number;
};

const NO_LIVE_ARTIFACTS: LiveArtifacts = { ids: [], revision: 0 };

/** The keystroke that folds the workspace, as it is shown beside the control. */
const WORKSPACE_SHORTCUT = "⌘\\";

export function useDesk(): DeskContextValue {
  const value = useContext(DeskContext);
  if (value === null) {
    throw new Error("useDesk must be used inside the Desk layout");
  }

  return value;
}

/**
 * The Desk itself: one room, the whole window — the rail down the left, the
 * conversation in the middle and what it produced on the right.
 *
 * The frame owns everything that outlives a single page inside it — the
 * conversations, the agents, the delete confirmation, and the state of the
 * rail and the workspace — because all the pages of the Desk share them and
 * none of them should reload when you move between them.
 *
 * It also owns the header, which means the header can say what the open
 * conversation is without the conversation having to draw a title bar of
 * its own. There is one strip at the top of the room, not one per column.
 */
export function DeskLayout({ activeThreadId }: { activeThreadId: string | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const wide = useMediaQuery(DESK_WIDE_QUERY);
  const setLastAgentId = useAssistantStore((state) => state.setLastAgentId);
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);
  const rail = useDeskStore((state) => state.rail);
  const setRail = useDeskStore((state) => state.setRail);
  const toggleRail = useDeskStore((state) => state.toggleRail);
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const togglePane = useDeskStore((state) => state.togglePane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);

  const [deleting, setDeleting] = useState<AssistantThread | null>(null);
  const [working, setWorking] = useState(false);
  const [liveArtifacts, setLiveArtifacts] = useState<LiveArtifacts>(NO_LIVE_ARTIFACTS);
  // On a narrow screen the rail is a sheet over the room rather than a column
  // beside it, and whether that sheet is open is this visit's business, not
  // a habit to remember.
  const [railSheetOpen, setRailSheetOpen] = useState(false);

  const threadsQuery = useQuery(queries.assistant.threads());
  const agentsQuery = useQuery(queries.assistant.myAgents());
  const threads = useMemo(() => threadsQuery.data?.items ?? [], [threadsQuery.data?.items]);
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  const listedThread = useMemo(
    () => threads.find((thread) => thread.id === activeThreadId) ?? null,
    [activeThreadId, threads],
  );
  // The list leaves out quick questions nobody kept, and a finished answer's
  // notice links to one. Read on its own only once the list has said it does
  // not have it, so a listed conversation never costs a second request.
  const unlistedQuery = useQuery({
    ...queries.assistant.thread(activeThreadId ?? ""),
    enabled: activeThreadId !== null && threadsQuery.isSuccess && listedThread === null,
    retry: false,
  });
  const activeThread =
    listedThread ??
    (activeThreadId !== null && unlistedQuery.data?.id === activeThreadId
      ? unlistedQuery.data
      : null);
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
    setLiveArtifacts(NO_LIVE_ARTIFACTS);
  }

  // A sheet left open while the window grows into a wide one would sit over
  // the rail it stands in for. Adjusted during render rather than in an
  // effect, so the first wide frame is already without it.
  const [seenWide, setSeenWide] = useState(wide);
  if (seenWide !== wide) {
    setSeenWide(wide);
    if (wide) {
      setRailSheetOpen(false);
    }
  }

  // An unlisted conversation is cached on its own, so a rename or a pin has
  // to refresh that copy as well as the list.
  const refreshThreads = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.thread._def }),
      ]),
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
      setRailSheetOpen(false);
      void navigate(conversationPath(thread.id));
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
  // folded away: the person asked for the thing it holds. Every change to the
  // set — a table growing, a card withdrawn — is a new revision, so the pane
  // reads the set again rather than only when a new id arrives.
  const noteLiveArtifacts = useCallback(
    (artifacts: readonly AssistantArtifactEvent[]) => {
      setLiveArtifacts((live) => ({
        ids: artifacts.map((artifact) => artifact.id),
        revision: live.revision + 1,
      }));
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
      activeThread,
      agents,
      agentsById,
      agentsUnavailable: agentsQuery.isError,
      isLoading: threadsQuery.isLoading || agentsQuery.isLoading || unlistedQuery.isLoading,
      isStarting: startMutation.isPending,
      start: (agentId, question) => startMutation.mutate({ agentId, question }),
      remove: setDeleting,
      togglePin: (thread) => pinMutation.mutate(thread),
      setWorking,
      noteLiveArtifacts,
      openArtifact,
    }),
    [
      activeThread,
      agents,
      agentsById,
      agentsQuery.isError,
      agentsQuery.isLoading,
      noteLiveArtifacts,
      openArtifact,
      pinMutation,
      startMutation,
      threads,
      threadsQuery.isLoading,
      unlistedQuery.isLoading,
    ],
  );

  const workspaceOpen = activeThread !== null && pane === "open";
  const railOpen = railIsOpen(rail);
  const workspaceSize = useDeskStore((state) => state.workspaceSize);
  const setWorkspaceSize = useDeskStore((state) => state.setWorkspaceSize);

  const showRail = useCallback(() => {
    if (wide) {
      setRail("open");
    } else {
      setRailSheetOpen(true);
    }
  }, [setRail, wide]);

  // The room's two shortcuts. ⌘B folds the rail, as it does the sidebar in
  // the rest of the app; ⌘\ folds the workspace. A person reading a wide
  // table wants the conversation out of the way and then wants it back a
  // sentence later, and reaching for a button in the corner each time is
  // the sort of friction that makes a workspace feel like a web page.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || !(event.metaKey || event.ctrlKey) || event.altKey) {
        return;
      }
      if (event.key === "b" || event.key === "B") {
        event.preventDefault();
        if (wide) {
          toggleRail();
        } else {
          setRailSheetOpen((open) => !open);
        }
      } else if (event.key === "\\" && activeThread !== null) {
        event.preventDefault();
        togglePane();
      } else if (event.key === "k" || event.key === "K") {
        const search = document.querySelector<HTMLInputElement>(
          '[data-slot="desk-rail"] input[type="text"], [data-slot="desk-rail"] input:not([type])',
        );
        if (search) {
          event.preventDefault();
          if (wide) {
            setRail("open");
          }
          search.focus();
          search.select();
        }
      }
    };
    window.addEventListener("keydown", onKeyDown);

    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeThread, togglePane, toggleRail, wide]);

  const railProps = {
    threads,
    agents,
    activeThreadId,
    isLoading: threadsQuery.isLoading || agentsQuery.isLoading,
    listUnavailable: threadsQuery.isError,
    isStarting: startMutation.isPending,
    onStart: (agentId: string) => startMutation.mutate({ agentId }),
    onDelete: setDeleting,
    onTogglePin: (thread: AssistantThread) => pinMutation.mutate(thread),
    onRetry: () => void threadsQuery.refetch(),
  };

  return (
    <DeskContext.Provider value={value}>
      <DeskShell
        rail={
          <DeskRailFold
            open={railOpen}
            rail={
              <DeskRail
                {...railProps}
                collapse={{ label: t("Hide the rail"), shortcut: RAIL_SHORTCUT }}
                onCollapse={() => setRail("collapsed")}
              />
            }
            strip={
              <DeskRailStrip
                threads={threads}
                isStarting={startMutation.isPending}
                onStart={(agentId) => startMutation.mutate({ agentId })}
                onExpand={() => setRail("open")}
              />
            }
          />
        }
        railOpen={railOpen}
        onShowRail={showRail}
        accent={accent}
        working={working}
        lead={
          activeThread ? (
            <span className="text-foreground-muted flex min-w-0 items-center gap-2 pl-1 text-sm">
              <AgentTile agent={activeAgent} size="sm" />
              <span className="hidden max-w-40 truncate sm:inline">
                {activeAgent?.name ?? t("Agent unavailable")}
              </span>
              <span aria-hidden className="text-foreground-subtle">
                /
              </span>
            </span>
          ) : undefined
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
            <span className="truncate px-1.5 text-sm font-medium">{pageName(t, pathname)}</span>
          )
        }
        actions={
          activeThread && (
            <>
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
              <span aria-hidden className="bg-desk-hairline mx-1 hidden h-5 w-px lg:block" />
              <HeaderAction
                label={pane === "open" ? t("Hide the workspace") : t("Show the workspace")}
                pressed={pane === "open"}
                hint={WORKSPACE_SHORTCUT}
                className="hidden lg:inline-flex"
                onClick={togglePane}
              >
                <PanelRightIcon className="size-4" />
              </HeaderAction>
            </>
          )
        }
      >
        <DeskColumns
          workspaceOpen={workspaceOpen}
          workspaceSize={workspaceSize}
          onWorkspaceResize={setWorkspaceSize}
          conversation={<Outlet />}
          workspace={
            activeThread ? (
              <DeskWorkspace
                key={activeThread.id}
                threadId={activeThread.id}
                agent={activeAgent}
                liveArtifacts={liveArtifacts}
                onClose={() => setPane("closed")}
                className="h-full"
              />
            ) : (
              <DeskWorkspaceEmpty />
            )
          }
        />
      </DeskShell>

      <Sheet open={railSheetOpen && !wide} onOpenChange={setRailSheetOpen}>
        <SheetContent
          side="left"
          showCloseButton={false}
          className={cn(
            "bg-desk-rail gap-0 rounded-none border-0 p-0",
            "data-[side=left]:w-[min(100vw_-_3rem,17rem)] data-[side=left]:sm:max-w-none",
          )}
        >
          <SheetTitle className="sr-only">{t("Conversations")}</SheetTitle>
          <DeskRail
            {...railProps}
            collapse={{ label: t("Close") }}
            onCollapse={() => setRailSheetOpen(false)}
            onNavigate={() => setRailSheetOpen(false)}
          />
        </SheetContent>
      </Sheet>

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

/** What the strip calls the page when no conversation is open. */
function pageName(t: ReturnType<typeof useT>, pathname: string): string {
  if (pathname.startsWith("/desk/decisions")) {
    return t("Decisions");
  }
  if (pathname.startsWith("/desk/watchtower")) {
    return t("Watchtower");
  }

  return t("Today");
}

function HeaderAction({
  label,
  hint,
  pressed,
  destructive = false,
  className,
  onClick,
  children,
}: {
  label: string;
  /** The keystroke that does the same thing, shown in the tooltip. */
  hint?: string;
  pressed?: boolean;
  destructive?: boolean;
  className?: string;
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
              className,
            )}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent side="bottom" className="flex items-center gap-2">
        {label}
        {hint && <Kbd>{hint}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );
}
