import { Metadata } from "@/components/metadata";
import { deskTitle } from "./top/desk-title";
import "../_styles/desk-motion.css";
import { ContextMenuHost } from "@trenova/shared/components/context-menu/context-menu-host";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import { deskAttention } from "@/lib/queries/desk-attention";
import { inbox } from "@/lib/queries/inbox";
import {
  cachedThread,
  countCachedThread,
  findCachedThread,
  patchCachedThread,
  removeCachedThread,
  takeCachedThread,
} from "@/lib/thread-list";
import { useDeskThreads } from "./use-desk-threads";
import { assistantService, downloadAssistantTranscript } from "@/services/assistant";
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
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { KeyboardShortcutsDialog } from "@/components/keyboard-shortcuts-dialog";
import { isTypingTarget } from "@/lib/dom";
import { useAppDialogsStore } from "@/stores/app-dialogs-store";
import { Outlet, useLocation, useNavigate, useSearchParams } from "react-router";
import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import {
  useStartConversation,
  type StartBinding,
} from "@/components/assistant/use-start-conversation";
import { forgetThreadSchedules } from "@/components/assistant/thread-schedules";
import { useLensCounts } from "./attention/use-lens-counts";
import type { DeskPlace } from "./rail/desk-rail";
import { RAIL_WAITING_CASES } from "./rail/open-work";
import { DeskRailThreads } from "./rail/rail-threads";
import {
  DeskSettingsDialog,
  deskSettingsClasses,
  isDeskSettingsSection,
  type DeskSettingsSection,
} from "./desk-settings";
import { DeskTop } from "./top/desk-top";
import { CommandPaletteMount } from "@/components/command-palette/command-palette-mount";
import { PAGE_TRANSITION_CLASS } from "./desk-motion";
import { caseOfThread } from "./record-case/case-subject";
import { railViewOf, TRIAGE_BACK_HOME, type TriageBack } from "./rail/desk-rail";
import { useGoToChords } from "@/components/navigation/use-go-to-chords";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { DeskMargin } from "./margin/desk-margin";

/** G, then a letter, goes to a place in the Desk; the app's own chords go to modules. */
/** How long the mouse rests on the left edge before a floating rail comes out. */
const RAIL_HOVER_OPEN_MS = 90;
/** How long the mouse is off a floating rail before it goes back. */
const RAIL_HOVER_CLOSE_MS = 260;

const DESK_CHORDS: ReadonlyMap<string, string> = new Map([
  ["D", "/desk"],
  ["N", "/desk/needs"],
  ["W", "/desk/needs?lens=watchtower"],
  ["T", "/desk/triage"],
]);

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
  /** Starts a conversation in Nova's margin, beside the page, instead of on a page of its own. */
  startInMargin: (
    agentId: string,
    question?: string,
    extras?: DeskStartExtras,
    binding?: StartBinding,
  ) => void;
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
  if (pathname.startsWith("/desk/needs")) {
    return "needs";
  }
  if (pathname.startsWith("/desk/r/")) {
    return "case";
  }
  if (pathname.startsWith("/desk/handled/")) {
    return "handled";
  }
  if (pathname.startsWith("/desk/v/")) {
    return "view";
  }
  if (pathname.startsWith("/desk/triage")) {
    return "triage";
  }
  if (pathname.startsWith("/desk/memory")) {
    return "memory";
  }
  if (pathname.startsWith("/desk/scheduled")) {
    return "scheduled";
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
  const { pathname, search } = useLocation();
  const pane = useDeskStore((state) => state.pane);
  const setPane = useDeskStore((state) => state.setPane);
  const setActiveArtifact = useDeskStore((state) => state.setActiveArtifact);
  const { allowed: canDecide } = usePermission(Resource.AgentProposal, Operation.Read);
  const { allowed: canWatch } = usePermission(Resource.Watchtower, Operation.Read);
  const { allowed: canTriage } = usePermission(Resource.InboundMessage, Operation.Read);
  const lensCounts = useLensCounts(canWatch);
  const { data: waiting } = useQuery({
    ...deskAttention.list({
      lens: "Decisions",
      kinds: ["AgentPlan", "AgentProposal"],
      first: RAIL_WAITING_CASES,
    }),
    enabled: canWatch && canDecide,
  });
  const { data: triageCounts } = useQuery({ ...inbox.counts(), enabled: canTriage });

  const [working, setWorking] = useState(false);
  const [liveArtifacts, setLiveArtifacts] = useState<LiveArtifacts>(NO_LIVE_ARTIFACTS);
  const [pendingLookups, setPendingLookups] = useState<ReadonlySet<string>>(NO_PENDING_LOOKUPS);
  const [artifactCount, setArtifactCount] = useState(0);
  const [newArtifact, setNewArtifact] = useState(false);
  // On a phone, and at any width when the rail floats, the rail slides over
  // the page instead of sitting beside it, and is put away again once a place
  // in it is chosen.
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
  // Beside the page the rail folds away and stays folded until it is opened
  // again; its sheet on a phone is never folded.
  const railFoldKept = useDeskSettingsStore((state) => state.railFolded);
  const setRailFolded = useDeskSettingsStore((state) => state.setRailFolded);
  const caseLabel = useDeskMarginStore((state) => state.casePage?.label ?? null);
  const deskPageTitle = useDeskMarginStore((state) => state.pageTitle);
  const railMode = useDeskSettingsStore((state) => state.settings.rail);
  const railFloats = railMode === "floating";
  // A floating rail comes out when the mouse reaches the left edge of the
  // screen and goes back once it has left the rail. Both wait a moment, so
  // brushing the edge does not open it and a slip off it does not close it.
  // Touch and pen never hover; they use the button and a tap outside.
  const railHover = useRef<number | null>(null);
  const railHoverSoon = (open: boolean, wait: number) => {
    if (railHover.current !== null) {
      window.clearTimeout(railHover.current);
    }
    railHover.current = window.setTimeout(() => {
      railHover.current = null;
      // A menu or dialog opened from the rail is drawn outside it: the rail
      // stays while one is up.
      if (!open && document.querySelector('[role="menu"], [role="dialog"]') !== null) {
        return;
      }
      setRailOpen(open);
    }, wait);
  };
  const railHoverStay = () => {
    if (railHover.current !== null) {
      window.clearTimeout(railHover.current);
      railHover.current = null;
    }
  };
  useEffect(() => railHoverStay, []);
  const railFolded = !railFloats && railFoldKept && !railOpen;
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
  // A conversation with an agent that runs on its own (the one working a
  // case) is named too: those are left out of the list of chat agents, so the
  // ones the conversations on screen need are read by id.
  const unnamedAgentIds = useMemo(() => {
    if (agentsQuery.data === undefined) {
      return [];
    }
    const known = new Set(agentsQuery.data.map((agent) => agent.id));
    const missing = new Set<string>();
    for (const thread of threads) {
      if (thread.agentDefinitionId !== "" && !known.has(thread.agentDefinitionId)) {
        missing.add(thread.agentDefinitionId);
      }
    }
    return [...missing].sort();
  }, [agentsQuery.data, threads]);
  const { data: namedAgents } = useQuery({
    ...assistant.agentChoicesByIds(unnamedAgentIds),
    enabled: unnamedAgentIds.length > 0,
    staleTime: 60_000,
  });
  const agentsById = useMemo(
    () => new Map([...agents, ...(namedAgents ?? [])].map((agent) => [agent.id, agent])),
    [agents, namedAgents],
  );

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
      activeThreadId !== null && threadsQuery.isSuccess && listedThread === null && !activeDeleting,
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
  // The same view the rail draws, so the page and the rail change view together.
  const view = railViewOf(place, activeThread !== null && caseOfThread(activeThread) !== null);
  // The page a person was on before Triage, kept while they are in it, so its
  // way back is theirs. Held in memory only: an address can name a record.
  const [triageBack, setTriageBack] = useState<TriageBack>(TRIAGE_BACK_HOME);
  const here = pathname + search;
  if (view !== "triage" && triageBack.to !== here) {
    setTriageBack({ view, to: here });
  }
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
  const startMarginConversation = useStartConversation({
    origin: "Desk",
    onStarted: (thread) => useDeskMarginStore.getState().showThread(thread.id),
  });
  const marginOpen = useDeskMarginStore((state) => state.margin !== null);
  // A conversation opened full width is the margin's conversation: the margin
  // folds rather than draw the same thread twice.
  const marginThreadId = useDeskMarginStore((state) => state.margin?.threadId ?? null);
  useEffect(() => {
    if (activeThreadId !== null && marginThreadId === activeThreadId) {
      useDeskMarginStore.getState().closeMargin();
    }
  }, [activeThreadId, marginThreadId]);

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
    mutationFn: (thread: AssistantThread) => assistantService.deleteThread(thread.id),
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
      // Its schedules were deleted with it.
      forgetThreadSchedules(queryClient, thread.id);
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
      assistantService.updateThread(thread.id, { title }),
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
      assistantService.updateThread(thread.id, { pinned }),
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
    mutationFn: (id: string) => assistantService.markThreadRead(id),
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

  // ⌘J is Nova everywhere: in a conversation it puts the caret in its box;
  // anywhere else it opens Nova's margin beside the page, on what the page is
  // about, and folds it again.
  const askNova = useCallback(() => {
    if (activeThreadId !== null) {
      document.querySelector<HTMLElement>('.dk-mainc .dk-cmp [role="textbox"]')?.focus();
      return;
    }
    useDeskMarginStore.getState().toggleMargin();
  }, [activeThreadId]);

  useGoToChords(DESK_CHORDS, { viewTransition: true });

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented) {
        return;
      }
      // Esc folds Nova's margin from anywhere outside a text box.
      if (
        event.key === "Escape" &&
        !isTypingTarget(event.target) &&
        useDeskMarginStore.getState().margin !== null
      ) {
        event.preventDefault();
        useDeskMarginStore.getState().closeMargin();
        return;
      }
      // ? lists the Desk's keys from anywhere outside a text box.
      if (
        event.key === "?" &&
        !event.metaKey &&
        !event.ctrlKey &&
        !event.altKey &&
        !isTypingTarget(event.target)
      ) {
        event.preventDefault();
        useAppDialogsStore.getState().setDialogOpen("shortcuts", true);
        return;
      }
      if (!(event.metaKey || event.ctrlKey) || event.altKey) {
        return;
      }
      const key = event.key.toLowerCase();
      if (key === "o" && event.shiftKey) {
        // Ctrl+N belongs to the browser, which opens a window and never hands
        // the key to the page. A new conversation opens in Nova's margin.
        event.preventDefault();
        if (useDeskMarginStore.getState().casePage !== null) {
          void navigate("/desk");
          return;
        }
        useDeskMarginStore.getState().openMargin(null);
      } else if (key === "\\" && useDeskMarginStore.getState().margin !== null) {
        event.preventDefault();
        useDeskMarginStore.getState().closeMargin();
      } else if (key === "\\" && activeThread !== null) {
        event.preventDefault();
        const { pane: current, browsing, setBrowsing } = useDeskStore.getState();
        const next = commandJ({ open: current === "open", browsing });
        setBrowsing(next.browsing);
        setPane(next.open ? "open" : "closed");
      } else if (key === "b" && !event.shiftKey) {
        event.preventDefault();
        const desk = useDeskSettingsStore.getState();
        if (desk.settings.rail === "floating") {
          setRailOpen((shown) => !shown);
        } else {
          desk.setRailFolded(!desk.railFolded);
        }
      } else if (key === "j") {
        event.preventDefault();
        askNova();
      }
    };
    window.addEventListener("keydown", onKeyDown);

    return () => window.removeEventListener("keydown", onKeyDown);
  }, [activeThread, askNova, navigate, setPane]);

  const value = useMemo<DeskContextValue>(
    () => ({
      threads,
      activeThread,
      agents,
      agentsById,
      agentsUnavailable: agentsQuery.isError,
      isLoading: threadsQuery.isLoading || agentsQuery.isLoading || unlistedQuery.isLoading,
      isStarting: startConversation.isStarting || startMarginConversation.isStarting,
      start: startConversation.start,
      startInMargin: startMarginConversation.start,
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
      openSearch: () => useCommandPaletteStore.getState().openOn("conversations"),
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
      startMarginConversation.isStarting,
      startMarginConversation.start,
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
      <Metadata
        title={deskTitle(t, {
          place,
          threadTitle: activeThread?.title ?? null,
          agentName: pageAgent?.name ?? null,
          caseLabel,
          pageTitle: deskPageTitle,
          lens: searchParams.get("lens"),
        })}
      />
      <div
        className={cn(
          "dsk",
          deskSettingsClasses(settings),
          railOpen && "dk-rail-open",
          marginOpen && "dk-w-mg-open",
        )}
        data-settings={settingsOpen || undefined}
      >
        <DeskRailThreads
          place={place}
          agentsById={agentsById}
          activeThreadId={activeThreadId}
          canWatch={canWatch}
          canDecide={canDecide}
          canTriage={canTriage}
          lensCounts={lensCounts}
          triageWaiting={triageCounts?.waiting ?? 0}
          triageBack={triageBack}
          waitingCases={waiting?.groups}
          onSetPinned={(thread, pinned) => pinMutation.mutate({ thread, pinned })}
          onRename={(thread, title) => renameMutation.mutate({ thread, title })}
          onDownload={(thread) => downloadAssistantTranscript(thread.id)}
          onDelete={(thread) => deleteMutation.mutate(thread)}
          onSearch={() => {
            setRailOpen(false);
            useCommandPaletteStore.getState().openOn("conversations");
          }}
          onSettings={() => {
            setRailOpen(false);
            setSettingsOpen(true);
          }}
          activeThread={activeThread}
          folded={railFolded}
          mode={railMode}
          open={railOpen}
          onPointerEnter={railFloats ? railHoverStay : undefined}
          onPointerLeave={
            railFloats
              ? (event) => {
                  if (event.pointerType === "mouse") {
                    railHoverSoon(false, RAIL_HOVER_CLOSE_MS);
                  }
                }
              : undefined
          }
          onFold={() => {
            if (railOpen || railFloats) {
              setRailOpen(false);
            } else {
              setRailFolded(true);
            }
          }}
        />
        {railFloats && !railOpen && (
          <div
            aria-hidden
            data-slot="rail-edge"
            className="fixed inset-y-0 left-0 z-39 w-2 max-[720px]:hidden"
            onPointerEnter={(event) => {
              if (event.pointerType === "mouse") {
                railHoverSoon(true, RAIL_HOVER_OPEN_MS);
              }
            }}
            onPointerLeave={railHoverStay}
          />
        )}
        {railOpen && (
          <Button
            variant="bare"
            size="bare"
            className={cn(
              "fixed inset-0 z-39",
              // A floating rail leaves the page readable under it: the page is
              // only covered so a click on it puts the rail away.
              railFloats
                ? "max-[720px]:animate-[dk-fadein_200ms_ease_both] max-[720px]:bg-scrim"
                : "hidden animate-[dk-fadein_200ms_ease_both] bg-scrim max-[720px]:block",
            )}
            aria-label={t("Close the menu")}
            onClick={() => setRailOpen(false)}
          />
        )}
        {/* A floating rail is out of the room's grid, so the page takes both columns. */}
        <div className={cn("dk-mainc", railFloats && "col-span-full")}>
          <DeskTop
            onOpenRail={() => setRailOpen(true)}
            canWatch={canWatch}
            needsYou={lensCounts.everything}
            onUnfoldRail={
              railFloats
                ? () => setRailOpen(true)
                : railFolded
                  ? () => setRailFolded(false)
                  : undefined
            }
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
            onDelete={(thread) => deleteMutation.mutate(thread)}
          />
          {/* The page's main landmark: one cell the page fills, named for the Desk's
              view so the page moves with the rail when the view changes. */}
          <main
            className={cn(
              "grid min-h-0 min-w-0 grid-cols-[minmax(0,1fr)] grid-rows-[minmax(0,1fr)]",
              PAGE_TRANSITION_CLASS[view],
            )}
          >
            <Outlet />
          </main>
          <KeyboardShortcutsDialog />
          <CommandPaletteMount />

          {settingsOpen && (
            <DeskSettingsDialog
              agents={agents}
              initialSection={settingsSection ?? "appearance"}
              onClose={() => setSettingsOpen(false)}
            />
          )}
        </div>
        <DeskMargin />
        <ContextMenuHost />
      </div>
    </DeskContext.Provider>
  );
}
