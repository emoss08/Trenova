import type { TurnHandoff } from "@/components/assistant/turn-stream";
import { create } from "zustand";
import { persist } from "zustand/middleware";

/** The artifacts pane: open beside the conversation, or folded away. */
export type DeskPaneState = "open" | "closed";

/** The rail: the Desk's table of contents, open along the left, or folded to nothing. */
export type DeskRailState = "open" | "closed";

interface DeskState {
  /** Whether the rail of conversations and places is showing along the left. */
  rail: DeskRailState;
  /** Whether the artifacts pane is showing beside a conversation. */
  pane: DeskPaneState;
  /** The artifact each conversation last had open, so returning to it reopens the same one. */
  activeArtifactByThread: Record<string, string>;
  /** The note about the terms above the composer has been read or closed. */
  termsSeen: boolean;
  /** The replies each conversation has pinned as chapters, in the order pinned. */
  chaptersByThread: Record<string, string[]>;
  /** Whether questions asked at the Desk carry the page the person came from. */
  sharePage: boolean;
  /** Whether the pane shows every artifact to search rather than the open one. Not kept. */
  browsing: boolean;
  /**
   * A question an artifact asked the agent on the person's behalf, such as
   * "create a shipment from this document", waiting for its conversation to
   * send it. Not kept.
   */
  asks: Record<string, string>;
  /**
   * What the reply being written in each conversation says about handing it
   * to another agent, so the hand-off menu can offer one before the reply is
   * saved. Only a conversation with a reply under way has an entry. Not kept.
   */
  liveHandoff: Record<string, TurnHandoff>;

  setRail: (rail: DeskRailState) => void;
  toggleRail: () => void;
  setPane: (pane: DeskPaneState) => void;
  togglePane: () => void;
  setActiveArtifact: (threadId: string, artifactId: string | null) => void;
  markTermsSeen: () => void;
  toggleChapter: (threadId: string, messageId: string) => void;
  setSharePage: (sharePage: boolean) => void;
  setBrowsing: (browsing: boolean) => void;
  askAgent: (threadId: string, question: string) => void;
  takeAsk: (threadId: string) => string | null;
  setLiveHandoff: (threadId: string, live: TurnHandoff | null) => void;
}

/**
 * The live hand-offs with one conversation's changed, or the same record when
 * nothing did, so a reader keyed on it is not woken for nothing.
 */
export function withLiveHandoff(
  current: Record<string, TurnHandoff>,
  threadId: string,
  live: TurnHandoff | null,
): Record<string, TurnHandoff> {
  const known = current[threadId];
  if (live === null) {
    if (known === undefined) {
      return current;
    }
    const { [threadId]: _ended, ...rest } = current;
    return rest;
  }
  if (known !== undefined && known.asked === live.asked && known.agentIds === live.agentIds) {
    return current;
  }
  return { ...current, [threadId]: live };
}

/** Remembered artifacts for the most recently visited conversations. */
const MAX_REMEMBERED_ARTIFACTS = 40;

/**
 * Remembers which artifact a conversation had open, forgetting the oldest
 * once too many are kept. The newest is the one just set, so the oldest key
 * is the first inserted.
 */
export function rememberActiveArtifact(
  current: Record<string, string>,
  threadId: string,
  artifactId: string | null,
): Record<string, string> {
  const entries = Object.entries(current).filter(([key]) => key !== threadId);
  if (artifactId !== null) {
    entries.push([threadId, artifactId]);
  }

  return Object.fromEntries(entries.slice(Math.max(0, entries.length - MAX_REMEMBERED_ARTIFACTS)));
}

export const useDeskStore = create<DeskState>()(
  persist(
    (set, get) => ({
      rail: "open",
      pane: "open",
      activeArtifactByThread: {},
      termsSeen: false,
      chaptersByThread: {},
      sharePage: true,
      browsing: false,
      asks: {},
      liveHandoff: {},

      setRail: (rail) => set({ rail }),
      toggleRail: () => set((state) => ({ rail: state.rail === "open" ? "closed" : "open" })),
      setPane: (pane) => set({ pane }),
      togglePane: () => set((state) => ({ pane: state.pane === "open" ? "closed" : "open" })),
      setActiveArtifact: (threadId, artifactId) =>
        set((state) => ({
          activeArtifactByThread: rememberActiveArtifact(
            state.activeArtifactByThread,
            threadId,
            artifactId,
          ),
        })),
      markTermsSeen: () => set({ termsSeen: true }),
      setSharePage: (sharePage) => set({ sharePage }),
      setBrowsing: (browsing) => set({ browsing }),
      askAgent: (threadId, question) =>
        set((state) => ({ asks: { ...state.asks, [threadId]: question } })),
      takeAsk: (threadId) => {
        const question = get().asks[threadId] ?? null;
        if (question !== null) {
          set((state) => {
            const { [threadId]: _taken, ...rest } = state.asks;
            return { asks: rest };
          });
        }
        return question;
      },
      setLiveHandoff: (threadId, live) => {
        const current = get().liveHandoff;
        const next = withLiveHandoff(current, threadId, live);
        if (next !== current) {
          set({ liveHandoff: next });
        }
      },
      toggleChapter: (threadId, messageId) =>
        set((state) => {
          const current = state.chaptersByThread[threadId] ?? [];
          const next = current.includes(messageId)
            ? current.filter((id) => id !== messageId)
            : [...current, messageId];
          return { chaptersByThread: { ...state.chaptersByThread, [threadId]: next } };
        }),
    }),
    {
      name: "trenova-desk",
      partialize: (state) => ({
        rail: state.rail,
        pane: state.pane,
        activeArtifactByThread: state.activeArtifactByThread,
        termsSeen: state.termsSeen,
        chaptersByThread: state.chaptersByThread,
        sharePage: state.sharePage,
      }),
    },
  ),
);
