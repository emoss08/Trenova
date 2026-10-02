import { useApiMutation } from "@/hooks/use-api-mutation";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifactEvent, AssistantThread } from "@/types/assistant";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import { DeskRail, type DeskPlace } from "./desk-rail";
import { DeskTopBar } from "./desk-topbar";

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
  working: boolean;
  /** Told what a streaming turn has produced so far, so the workspace opens on the newest and follows the set. */
  noteLiveArtifacts: (artifacts: readonly AssistantArtifactEvent[]) => void;
  liveArtifacts: LiveArtifacts;
  /** Opens an artifact the transcript referred to. */
  openArtifact: (threadId: string, artifactId: string) => void;
  /** Whether the workspace beside the open conversation is showing. */
  workspaceOpen: boolean;
  setWorkspaceOpen: (open: boolean) => void;
  /** How many artifacts the open conversation holds, for the workspace button. */
  setArtifactCount: (count: number) => void;
  openSearch: () => void;
  openSettings: () => void;
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

export function useDesk(): DeskContextValue {
  const value = useContext(DeskContext);
  if (value === null) {
    throw new Error("useDesk must be used inside the Desk layout");
  }

  return value;
}

function placeFor(pathname: string, activeThreadId: string | null): DeskPlace {
  if (activeThreadId !== null) {
    return "thread";
  }
  if (pathname.startsWith("/desk/decisions")) {
    return "decisions";
  }
  if (pathname.startsWith("/desk/watchtower")) {
    return "watchtower";
  }

  return "today";
}

/**
 * The Desk itself: one room, the whole window — the rail down the left and
 * the page beside it under one strip that says where you are.
 *
 * The frame owns everything that outlives a single page inside it — the
 * conversations, the agents, and the state of the workspace — because all
 * the pages of the Desk share them and none of them should reload when you
 * move between them.
 */
export function DeskLayout({ activeThreadId }: { activeThreadId: string | null }) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const { pathname } = useLocation();
  const setLastAgentId = useAssistantStore((state) => state.setLastAgentId);
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const togglePane = useDeskStore((state) => state.togglePane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const { data: watchtowerCounts } = useQuery({
    ...queries.watchtower.counts(),
    enabled: canWatch,
  });

  const [working, setWorking] = useState(false);
  const [liveArtifacts, setLiveArtifacts] = useState<LiveArtifacts>(NO_LIVE_ARTIFACTS);
  const [artifactCount, setArtifactCount] = useState(0);
  const [newArtifact, setNewArtifact] = useState(false);
  const [searching, setSearching] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

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
  const place = placeFor(pathname, activeThreadId);

  // Leaving a conversation leaves its turn behind with it. Adjusted during
  // render rather than in an effect, so the first frame of a new conversation
  // is already dark.
  const [seenThreadId, setSeenThreadId] = useState(activeThreadId);
  if (seenThreadId !== activeThreadId) {
    setSeenThreadId(activeThreadId);
    setWorking(false);
    setLiveArtifacts(NO_LIVE_ARTIFACTS);
    setNewArtifact(false);
    setArtifactCount(0);
  }

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
      void navigate(conversationPath(thread.id));
    },
    resourceName: "Conversation",
  });

  const deleteMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.deleteThread(id),
    onSuccess: async (_result, id) => {
      toast.success(t("Conversation deleted"));
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

  // Opening a conversation reads it, and so does a reply landing while it is
  // open; the rail's dot for an unseen reply clears either way.
  const readMutation = useApiMutation({
    mutationFn: (id: string) => apiService.assistantService.markThreadRead(id),
    onSuccess: refreshThreads,
    resourceName: "Conversation",
  });
  const unreadHere = place === "thread" && activeThread?.attention?.unread === true;
  const readThreadId = unreadHere ? activeThread.id : null;
  const { mutate: markRead, isPending: marking } = readMutation;
  useEffect(() => {
    if (readThreadId !== null && !marking) {
      markRead(readThreadId);
    }
  }, [markRead, marking, readThreadId]);

  // A turn that produces something opens the workspace on it, even if it was
  // folded away: the person asked for the thing it holds.
  const noteLiveArtifacts = useCallback(
    (artifacts: readonly AssistantArtifactEvent[]) => {
      setLiveArtifacts((live) => ({
        ids: artifacts.map((artifact) => artifact.id),
        revision: live.revision + 1,
      }));
      setNewArtifact(true);
    },
    [],
  );

  const openArtifact = useCallback(
    (threadId: string, artifactId: string) => {
      setActiveArtifact(threadId, artifactId);
      setPane("open");
      setNewArtifact(false);
    },
    [setActiveArtifact, setPane],
  );

  const workspaceOpen = activeThread !== null && pane === "open";
  const setWorkspaceOpen = useCallback(
    (open: boolean) => {
      setPane(open ? "open" : "closed");
      if (open) {
        setNewArtifact(false);
      }
    },
    [setPane],
  );

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented || !(event.metaKey || event.ctrlKey) || event.altKey) {
        return;
      }
      const key = event.key.toLowerCase();
      if (key === "k") {
        event.preventDefault();
        setSearching((open) => !open);
      } else if (key === "n" && !event.shiftKey) {
        event.preventDefault();
        void navigate("/desk");
      } else if (key === "\\" && activeThread !== null) {
        event.preventDefault();
        togglePane();
      }
    };
    window.addEventListener("keydown", onKeyDown);

    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeThread, navigate, togglePane]);

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
      remove: (thread) => deleteMutation.mutate(thread.id),
      togglePin: (thread) => pinMutation.mutate(thread),
      setWorking,
      working,
      noteLiveArtifacts,
      liveArtifacts,
      openArtifact,
      workspaceOpen,
      setWorkspaceOpen,
      setArtifactCount,
      openSearch: () => setSearching(true),
      openSettings: () => setSettingsOpen(true),
    }),
    [
      activeThread,
      agents,
      agentsById,
      agentsQuery.isError,
      agentsQuery.isLoading,
      deleteMutation,
      liveArtifacts,
      noteLiveArtifacts,
      openArtifact,
      pinMutation,
      setWorkspaceOpen,
      startMutation,
      threads,
      threadsQuery.isLoading,
      unlistedQuery.isLoading,
      working,
      workspaceOpen,
    ],
  );

  const pendingHere = (activeThread?.attention?.pendingDecisions ?? 0) > 0;

  return (
    <DeskContext.Provider value={value}>
      <div className="dsk" data-searching={searching || undefined} data-settings={settingsOpen || undefined}>
        <DeskRail
          place={place}
          threads={threads}
          agentsById={agentsById}
          activeThreadId={activeThreadId}
          canWatch={canWatch}
          canDecide={canDecide}
          watchtowerCount={watchtowerCounts?.unresolved ?? 0}
          decisionsCount={attention?.agentDecisions ?? 0}
          decisionsWaitHere={pendingHere}
          onSearch={() => setSearching(true)}
          onSettings={() => setSettingsOpen(true)}
          onTogglePin={(thread) => pinMutation.mutate(thread)}
          onDelete={(thread) => deleteMutation.mutate(thread.id)}
        />
        <div className="dk-mainc">
          <DeskTopBar
            place={place}
            thread={activeThread}
            agent={activeAgent}
            workspaceOpen={workspaceOpen}
            artifactCount={artifactCount}
            newArtifact={newArtifact}
            pending={pendingHere}
            onToggleWorkspace={() => setWorkspaceOpen(!workspaceOpen)}
            onTogglePin={() => activeThread && pinMutation.mutate(activeThread)}
            onDownload={() => activeThread && downloadAssistantTranscript(activeThread.id)}
          />
          <Outlet />
        </div>
      </div>
    </DeskContext.Provider>
  );
}
