import { useApiMutation } from "@/hooks/use-api-mutation";
import { useAttentionSummary } from "@/hooks/use-attention";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { watchtower } from "@/lib/queries/watchtower";
import {
  cachedThread,
  countCachedThread,
  findCachedThread,
  patchCachedThread,
  removeCachedThread,
  takeCachedThread,
} from "@/lib/thread-list";
import { useDeskThreads } from "./use-desk-threads";
import { apiService } from "@/services/api";
import { downloadAssistantTranscript } from "@/services/assistant";
import { useDeskSettingsStore } from "@/stores/desk-settings-store";
import { commandJ } from "./artifacts/desk-workspace-state";
import { NO_PENDING_LOOKUPS, pendingLookupIds } from "./artifacts/pending-lookups";
import { useDeskStore } from "@/stores/desk-store";
import {
  LOOKUP_ARTIFACT_KINDS,
  type AssistantArtifactEvent,
  type AssistantThread,
} from "@/types/assistant";
import { hashKey, useMutationState, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Outlet, useLocation, useNavigate, useSearchParams } from "react-router";
import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import { useStartConversation } from "@/components/assistant/use-start-conversation";
import type { DeskPlace } from "./desk-rail";
import { DeskRailThreads } from "./desk-rail-threads";
import { DeskSearchPalette } from "./desk-search";
import {
  DeskSettingsDialog,
  deskSettingsClasses,
  isDeskSettingsSection,
  type DeskSettingsSection,
} from "./desk-settings";
import { DeskTopBar } from "./desk-topbar";

export type DeskContextValue = {
  /**
   * The person's most recent conversations (the list's first page), for what
   * reads recency, such as which agents to offer first. The same array until
   * that page changes: older pages the rail reads never touch it.
   */
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
  /** The open conversation is being deleted, and the Desk is moving off it. */
  deletingActive: boolean;
  setPinned: (thread: AssistantThread, pinned: boolean) => void;
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

/** Every delete of a conversation from the Desk, so the frame knows which are on their way out. */
const DELETE_THREAD_KEY = ["assistant-thread-delete"];

/**
 * Pins and renames go to the server one at a time, in the order they were
 * asked for, so a quick second toggle can never land before the first.
 */
const PIN_THREAD_SCOPE = { id: "assistant-thread-pin" };
const RENAME_THREAD_SCOPE = { id: "assistant-thread-rename" };

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
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const togglePane = useDeskStore((state) => state.togglePane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { data: attention } = useAttentionSummary();
  const { data: watchtowerCounts } = useQuery({
    ...watchtower.counts(),
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
  // Settings open on the section a link asks for (`/desk?settings=checklists`
  // is where the retired case checklist page now sends people), else on the
  // first. The request is taken off the address once it has opened them.
  const [searchParams, setSearchParams] = useSearchParams();
  const requestedSettings = searchParams.get("settings");
  const [settingsSection, setSettingsSection] = useState<DeskSettingsSection | null>(null);
  if (requestedSettings !== null && settingsSection === null) {
    setSettingsSection(isDeskSettingsSection(requestedSettings) ? requestedSettings : "appearance");
  }
  useEffect(() => {
    if (requestedSettings === null) return;
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete("settings");
        return next;
      },
      { replace: true },
    );
  }, [requestedSettings, setSearchParams]);
  const settingsOpen = settingsSection !== null;
  const setSettingsOpen = useCallback(
    (open: boolean) => setSettingsSection(open ? "appearance" : null),
    [setSettingsSection],
  );
  const settings = useDeskSettingsStore((state) => state.settings);

  const threadsQuery = useDeskThreads(activeThreadId);
  const agentsQuery = useQuery(assistant.myAgents());
  const { threads, listedThread } = threadsQuery;
  const agents = useMemo(() => agentsQuery.data ?? [], [agentsQuery.data]);
  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);

  // A conversation being deleted leaves the lists before the Desk has moved
  // off it; it is neither read on its own nor drawn as open meanwhile.
  const deletingIds = useMutationState({
    filters: { mutationKey: DELETE_THREAD_KEY, status: "pending" },
    select: (mutation) => (mutation.state.variables as AssistantThread | undefined)?.id,
  });
  const activeDeleting = activeThreadId !== null && deletingIds.includes(activeThreadId);

  // The list's first page leaves out older conversations, which the rail's
  // shelves read on their own, and quick questions nobody kept, which a
  // finished answer's notice links to. Read on its own only once the list has
  // said it does not have it, drawn meanwhile from whichever shelf holds it.
  const unlistedQuery = useQuery({
    ...assistant.thread(activeThreadId ?? ""),
    enabled:
      activeThreadId !== null &&
      threadsQuery.isSuccess &&
      listedThread === null &&
      !activeDeleting,
    placeholderData: () => findCachedThread(queryClient, activeThreadId) ?? undefined,
    retry: false,
  });
  const activeThread = activeDeleting
    ? null
    : (listedThread ??
      (activeThreadId !== null && unlistedQuery.data?.id === activeThreadId
        ? unlistedQuery.data
        : null));
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
        queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey }),
        queryClient.invalidateQueries({ queryKey: assistant.thread._def }),
      ]),
    [queryClient],
  );

  const startConversation = useStartConversation({
    origin: "Desk",
    onStarted: (thread) => void navigate(conversationPath(thread.id)),
  });

  // Every change to a conversation shows the moment it is asked for: reads
  // under way are stopped first so none of them writes the old state back,
  // and a refused change is taken out again. A stopped read puts its query
  // back as it was there and then, so the change is written straight after,
  // in the same task as the click: awaiting the stop would draw the old state
  // for a frame first.
  const holdThreadReads = useCallback(
    (threadId: string) => {
      void queryClient.cancelQueries({ queryKey: assistant.threads().queryKey });
      void queryClient.cancelQueries({ queryKey: assistant.thread(threadId).queryKey });
    },
    [queryClient],
  );

  const deleteMutation = useApiMutation({
    mutationKey: DELETE_THREAD_KEY,
    mutationFn: (thread: AssistantThread) => apiService.assistantService.deleteThread(thread.id),
    onMutate: (thread) => {
      if (activeThreadId === thread.id) {
        void navigate("/desk");
      }
      holdThreadReads(thread.id);
      // Out of every page at once: a refresh reads only the pages in view, and
      // the conversation must not linger in one that is not.
      const taken = takeCachedThread(queryClient, thread.id);
      const counted = taken.thread ?? thread;
      countCachedThread(queryClient, counted, -1);

      return {
        restore: () => {
          taken.restore();
          countCachedThread(queryClient, counted, 1);
        },
      };
    },
    onSuccess: (_result, thread) => {
      // A read that crossed the delete may have put it back.
      removeCachedThread(queryClient, thread.id);
      queryClient.removeQueries({ queryKey: assistant.thread(thread.id).queryKey, exact: true });
    },
    onError: (_error, _thread, context) => context?.restore(),
    onSettled: refreshThreads,
    resourceName: "Conversation",
  });

  // A new name moves nothing in the list, so it is written into the cached
  // conversation as it is sent rather than read back with every page.
  const renameMutation = useApiMutation({
    scope: RENAME_THREAD_SCOPE,
    mutationFn: ({ thread, title }: { thread: AssistantThread; title: string }) =>
      apiService.assistantService.updateThread(thread.id, { title }),
    onMutate: ({ thread, title }) => {
      holdThreadReads(thread.id);
      return { undo: patchCachedThread(queryClient, thread.id, { title }) };
    },
    onSuccess: (updated, { title }) => {
      // A later rename already on its way is what the person sees, and its
      // own answer is written when it comes.
      if (cachedThread(queryClient, updated.id)?.title !== title) {
        return;
      }
      patchCachedThread(queryClient, updated.id, {
        title: updated.title,
        version: updated.version,
        updatedAt: updated.updatedAt,
      });
      // A read already under way was asked before the rename and would put
      // the old name back; it is asked again instead.
      if (queryClient.isFetching({ queryKey: assistant.threads().queryKey }) > 0) {
        void queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
      }
    },
    onError: (_error, _variables, context) => {
      context?.undo();
      void refreshThreads();
    },
    resourceName: "Conversation",
  });

  // A pin moves the conversation between the pinned shelf and its dated one
  // at once; the shelves and their counts are read again once it settles.
  // Each pin names the state it asks for rather than flipping the one it was
  // given, so a second click made before the first reaches every reader
  // still undoes it.
  const pinMutation = useApiMutation({
    scope: PIN_THREAD_SCOPE,
    mutationFn: ({ thread, pinned }: { thread: AssistantThread; pinned: boolean }) =>
      apiService.assistantService.updateThread(thread.id, { pinned }),
    onMutate: ({ thread, pinned }) => {
      holdThreadReads(thread.id);
      const before = cachedThread(queryClient, thread.id) ?? thread;
      if (before.pinned === pinned) {
        return { undo: () => undefined };
      }
      const moved = { ...before, pinned };
      const undo = patchCachedThread(queryClient, thread.id, { pinned });
      countCachedThread(queryClient, before, -1);
      countCachedThread(queryClient, moved, 1);

      return {
        undo: () => {
          undo();
          countCachedThread(queryClient, moved, -1);
          countCachedThread(queryClient, before, 1);
        },
      };
    },
    onSuccess: (updated, { pinned }) => {
      // A second click already on its way is what the person sees.
      if (cachedThread(queryClient, updated.id)?.pinned !== pinned) {
        return;
      }
      patchCachedThread(queryClient, updated.id, {
        pinned: updated.pinned,
        version: updated.version,
        updatedAt: updated.updatedAt,
      });
    },
    onError: (_error, _variables, context) => context?.undo(),
    onSettled: refreshThreads,
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
    const queryHash = hashKey(assistant.artifacts(activeThreadId)._ctx.summary.queryKey);
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
      isStarting: startConversation.isStarting,
      start: startConversation.start,
      remove: (thread) => deleteMutation.mutate(thread),
      deletingActive: activeDeleting,
      setPinned: (thread, pinned) => pinMutation.mutate({ thread, pinned }),
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
      activeDeleting,
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
      setSettingsOpen,
      setWorkspaceOpen,
      startConversation.isStarting,
      startConversation.start,
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
        <DeskRailThreads
          place={place}
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
          onSetPinned={(thread, pinned) => pinMutation.mutate({ thread, pinned })}
          onDelete={(thread) => deleteMutation.mutate(thread)}
          activeThread={activeThread}
        />
        {railOpen && (
          <Button
            variant="bare"
            size="bare"
            className="dk-rail-scrim fixed inset-0 z-39 hidden max-[720px]:block"
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
            onSetPinned={(pinned) =>
              activeThread && pinMutation.mutate({ thread: activeThread, pinned })
            }
            onDownload={() => activeThread && downloadAssistantTranscript(activeThread.id)}
            onRename={(title) =>
              activeThread && renameMutation.mutate({ thread: activeThread, title })
            }
          />
          <Outlet />
          {/* Over the main column only, as designed: the sidebar stays in view. */}
          {searching && (
            <DeskSearchPalette agentsById={agentsById} onClose={() => setSearching(false)} />
          )}
          {settingsOpen && (
            <DeskSettingsDialog
              agents={agents}
              initialSection={settingsSection ?? "appearance"}
              onClose={() => setSettingsOpen(false)}
            />
          )}
        </div>
      </div>
    </DeskContext.Provider>
  );
}
