import { useApiMutation } from "@/hooks/use-api-mutation";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { commandJ } from "./artifacts/desk-workspace-state";
import { NO_PENDING_LOOKUPS, pendingLookupIds } from "./artifacts/pending-lookups";
import { useDeskStore } from "@/stores/desk-store";
import {
  LOOKUP_ARTIFACT_KINDS,
  type AssistantArtifactEvent,
  type AssistantThread,
} from "@/types/assistant";
import { hashKey, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import type { DeskStartExtras } from "./desk-home";
import { DeskRail, type DeskPlace } from "./desk-rail";
import { DeskSearchPalette } from "./desk-search";
import { DeskSettingsDialog, deskSettingsClasses } from "./desk-settings";
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
  start: (agentId: string, question?: string, extras?: DeskStartExtras) => void;
  remove: (thread: AssistantThread) => void;
  togglePin: (thread: AssistantThread) => void;
  /** Told while a turn is running, so the room can light up for it. */
  setWorking: (working: boolean) => void;
  working: boolean;
  /** Told what a streaming turn has produced so far, so the workspace opens on the newest and follows the set. */
  noteLiveArtifacts: (artifacts: readonly AssistantArtifactEvent[]) => void;
  liveArtifacts: LiveArtifacts;
  /**
   * The lookups the running turn has saved, which the server drops unless the
   * finished reply points to them. Left out of the workspace and its counts
   * until the artifacts are read again after the turn, so none blinks in and
   * out of the list while the reply is written.
   */
  pendingLookups: ReadonlySet<string>;
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

/** How long a finished turn's lookups stay hidden if the artifacts are not read again. */
const LOOKUP_SETTLE_MS = 15_000;

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
  if (pathname.startsWith("/desk/memory")) {
    return "memory";
  }
  if (agentPageId(pathname) !== null) {
    return "agent";
  }

  return "today";
}

/** The agent whose capabilities page is open, from `/desk/agents/:agentId`. */
function agentPageId(pathname: string): string | null {
  const match = /^\/desk\/agents\/([^/]+)/u.exec(pathname);
  return match ? decodeURIComponent(match[1]) : null;
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
  const setHandoff = useDeskHandoffStore((state) => state.setHandoff);
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
  const [pendingLookups, setPendingLookups] = useState<ReadonlySet<string>>(NO_PENDING_LOOKUPS);
  const [artifactCount, setArtifactCount] = useState(0);
  const [newArtifact, setNewArtifact] = useState(false);
  const [searching, setSearching] = useState(false);
  // On a phone the rail slides over the page instead of sitting beside it,
  // and folds away again once a place in it is chosen.
  const [railOpen, setRailOpen] = useState(false);
  const [railPath, setRailPath] = useState(pathname);
  useEffect(() => {
    if (!railOpen) return;
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") setRailOpen(false);
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [railOpen]);
  if (railPath !== pathname) {
    setRailPath(pathname);
    setRailOpen(false);
  }
  const [settingsOpen, setSettingsOpen] = useState(false);
  const settings = useDeskSettingsStore((state) => state.settings);

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
  const pageAgentId = agentPageId(pathname);
  const pageAgent = pageAgentId !== null ? (agentsById.get(pageAgentId) ?? null) : null;

  // Leaving a conversation leaves its turn behind with it. Adjusted during
  // render rather than in an effect, so the first frame of a new conversation
  // is already dark.
  const [seenThreadId, setSeenThreadId] = useState(activeThreadId);
  if (seenThreadId !== activeThreadId) {
    setSeenThreadId(activeThreadId);
    setWorking(false);
    setLiveArtifacts(NO_LIVE_ARTIFACTS);
    setPendingLookups(NO_PENDING_LOOKUPS);
    setNewArtifact(false);
    setArtifactCount(0);
  }

  // Coming into the Desk, someone who asked to pick up where they left off
  // lands in their latest conversation. Only on the way in: the Today link
  // still leads to Today afterwards.
  const [entered, setEntered] = useState(activeThreadId !== null || pathname !== "/desk");
  if (!entered && threadsQuery.isSuccess) {
    setEntered(true);
    const latest = threads.reduce<AssistantThread | null>(
      (best, thread) =>
        best === null || thread.lastMessageAt > best.lastMessageAt ? thread : best,
      null,
    );
    if (settings.start === "last" && latest !== null) {
      void navigate(conversationPath(latest.id), { replace: true });
    }
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
    mutationFn: ({ agentId }: { agentId: string; question?: string; extras?: DeskStartExtras }) =>
      apiService.assistantService.startThread(agentId, { origin: "Desk" }),
    onSuccess: async (thread, { agentId, question, extras }) => {
      setLastAgentId(agentId);
      if (extras && (extras.files.length > 0 || extras.mentions.length > 0 || extras.providerId)) {
        setHandoff({
          threadId: thread.id,
          files: extras.files,
          mentions: extras.mentions,
          providerId: extras.providerId,
        });
      }
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

  const renameMutation = useApiMutation({
    mutationFn: ({ thread, title }: { thread: AssistantThread; title: string }) =>
      apiService.assistantService.updateThread(thread.id, { title }),
    onSuccess: refreshThreads,
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
  // Unless the person would rather open it themselves, in which case the
  // top bar only marks that something arrived.
  // A lookup's table or card is not among them: it is kept only if the
  // reply points to it, and it opens from the reply where it is named.
  const noteLiveArtifacts = useCallback(
    (artifacts: readonly AssistantArtifactEvent[]) => {
      setPendingLookups((current) => pendingLookupIds(artifacts, current));
      // Nor is a move to another page: the app follows it, and the reply
      // names the page as a link.
      const made = artifacts.filter(
        (artifact) => !LOOKUP_ARTIFACT_KINDS.has(artifact.kind) && artifact.kind !== "navigation",
      );
      setLiveArtifacts((live) => ({
        ids: made.map((artifact) => artifact.id),
        revision: live.revision + 1,
      }));
      if (made.length === 0) {
        return;
      }
      const { autoOpen, artNotify } = useDeskSettingsStore.getState().settings;
      if (autoOpen === "on") {
        setPane("open");
      }
      setNewArtifact(artNotify === "on");
    },
    [setPane],
  );

  // Once the turn is over the server has dropped the lookups the reply did
  // not point to; the next read of the artifacts after that holds only the
  // kept ones, and from then on they show like any other.
  useEffect(() => {
    if (working || pendingLookups.size === 0 || activeThreadId === null) {
      return;
    }
    const cache = queryClient.getQueryCache();
    const queryHash = hashKey(queries.assistant.artifacts(activeThreadId).queryKey);
    const seen = cache.get(queryHash)?.state.dataUpdateCount ?? 0;
    const settle = () => setPendingLookups(NO_PENDING_LOOKUPS);
    const timer = window.setTimeout(settle, LOOKUP_SETTLE_MS);
    const unsubscribe = cache.subscribe((event) => {
      const { query } = event;
      if (
        query.queryHash === queryHash &&
        query.state.dataUpdateCount > seen &&
        query.state.fetchStatus === "idle" &&
        !query.state.isInvalidated
      ) {
        settle();
      }
    });
    return () => {
      window.clearTimeout(timer);
      unsubscribe();
    };
  }, [activeThreadId, pendingLookups, queryClient, working]);

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
      } else if (key === "j" && activeThread !== null) {
        event.preventDefault();
        const { pane: current, browsing, setBrowsing } = useDeskStore.getState();
        const next = commandJ({ open: current === "open", browsing });
        setBrowsing(next.browsing);
        setPane(next.open ? "open" : "closed");
      }
    };
    window.addEventListener("keydown", onKeyDown);

    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeThread, navigate, setPane, togglePane]);

  const value = useMemo<DeskContextValue>(
    () => ({
      threads,
      activeThread,
      agents,
      agentsById,
      agentsUnavailable: agentsQuery.isError,
      isLoading: threadsQuery.isLoading || agentsQuery.isLoading || unlistedQuery.isLoading,
      isStarting: startMutation.isPending,
      start: (agentId, question, extras) => startMutation.mutate({ agentId, question, extras }),
      remove: (thread) => deleteMutation.mutate(thread.id),
      togglePin: (thread) => pinMutation.mutate(thread),
      setWorking,
      working,
      noteLiveArtifacts,
      liveArtifacts,
      pendingLookups,
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
      pendingLookups,
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
      <div
        className={cn("dsk", deskSettingsClasses(settings), railOpen && "dk-rail-open")}
        data-searching={searching || undefined}
        data-settings={settingsOpen || undefined}
      >
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
          onSearch={() => {
            setRailOpen(false);
            setSearching(true);
          }}
          onSettings={() => {
            setRailOpen(false);
            setSettingsOpen(true);
          }}
          onTogglePin={(thread) => pinMutation.mutate(thread)}
          onDelete={(thread) => deleteMutation.mutate(thread.id)}
          onRename={(thread, title) => renameMutation.mutate({ thread, title })}
        />
        {railOpen && (
          <button
            type="button"
            className="dk-rail-scrim"
            aria-label={t("Close the menu")}
            onClick={() => setRailOpen(false)}
          />
        )}
        <div className="dk-mainc">
          <DeskTopBar
            onOpenRail={() => setRailOpen(true)}
            place={place}
            thread={activeThread}
            agent={place === "agent" ? pageAgent : activeAgent}
            agents={agents}
            onOpenAgent={() => {
              if (activeThread && activeAgent) {
                void navigate(
                  `/desk/agents/${encodeURIComponent(activeAgent.id)}?from=${encodeURIComponent(activeThread.id)}`,
                );
              }
            }}
            workspaceOpen={workspaceOpen}
            artifactCount={artifactCount}
            newArtifact={newArtifact}
            pending={pendingHere}
            onToggleWorkspace={() => setWorkspaceOpen(!workspaceOpen)}
            onTogglePin={() => activeThread && pinMutation.mutate(activeThread)}
            onDownload={() => activeThread && downloadAssistantTranscript(activeThread.id)}
          />
          <Outlet />
          {/* Over the main column only, as designed: the sidebar stays in view. */}
          {searching && (
            <DeskSearchPalette agentsById={agentsById} onClose={() => setSearching(false)} />
          )}
          {settingsOpen && (
            <DeskSettingsDialog agents={agents} onClose={() => setSettingsOpen(false)} />
          )}
        </div>
      </div>
    </DeskContext.Provider>
  );
}
