import { create } from "zustand";
import { persist } from "zustand/middleware";

/** The artifacts pane: open beside the conversation, or folded away. */
export type DeskPaneState = "open" | "closed";

/**
 * A question asked on the front page, waiting for the conversation it
 * started to open and send it.
 *
 * Starting a thread and sending its first message are two requests, and the
 * navigation between them happens in the middle. This carries the question
 * across that gap so a person who types at the Desk's front door lands in a
 * conversation that is already answering, rather than one with their words
 * sitting unsent in the box.
 */
export type DeskOpeningQuestion = {
  threadId: string;
  text: string;
};

interface DeskState {
  /** Whether the artifacts pane is showing beside a conversation. */
  pane: DeskPaneState;
  /** The artifact each conversation last had open, so returning to it reopens the same one. */
  activeArtifactByThread: Record<string, string>;
  /** The agent the person last started a conversation with from the Desk. */
  lastAgentId: string | null;
  /** A question asked before its conversation existed. Never persisted. */
  openingQuestion: DeskOpeningQuestion | null;

  setPane: (pane: DeskPaneState) => void;
  togglePane: () => void;
  setActiveArtifact: (threadId: string, artifactId: string | null) => void;
  setLastAgentId: (agentId: string | null) => void;
  setOpeningQuestion: (question: DeskOpeningQuestion | null) => void;
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
      pane: "open",
      activeArtifactByThread: {},
      lastAgentId: null,
      openingQuestion: null,

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
      setLastAgentId: (agentId) => set({ lastAgentId: agentId }),
      setOpeningQuestion: (question) => set({ openingQuestion: question }),
    }),
    {
      name: "trenova-desk",
      partialize: (state) => ({
        pane: state.pane,
        activeArtifactByThread: state.activeArtifactByThread,
        lastAgentId: state.lastAgentId,
      }),
    },
  ),
);
