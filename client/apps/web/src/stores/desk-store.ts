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

  setRail: (rail: DeskRailState) => void;
  toggleRail: () => void;
  setPane: (pane: DeskPaneState) => void;
  togglePane: () => void;
  setActiveArtifact: (threadId: string, artifactId: string | null) => void;
  markTermsSeen: () => void;
  toggleChapter: (threadId: string, messageId: string) => void;
  setSharePage: (sharePage: boolean) => void;
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
    (set) => ({
      rail: "open",
      pane: "open",
      activeArtifactByThread: {},
      termsSeen: false,
      chaptersByThread: {},
      sharePage: true,

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
