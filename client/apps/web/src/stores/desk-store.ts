import { create } from "zustand";
import { persist } from "zustand/middleware";

/** The artifacts pane: open beside the conversation, or folded away. */
export type DeskPaneState = "open" | "closed";

/**
 * The rail: the Desk's table of contents, open along the left, or folded to a
 * strip of its places. "closed" is the old name of the strip, kept so a
 * remembered fold still reads.
 */
export type DeskRailState = "open" | "collapsed" | "closed";

/** What the workspace beside a conversation shows. */
export type DeskWorkspaceTab = "artifacts" | "decisions" | "activity";

/** The workspace's share of the room, in percent of the columns' width. */
export const DEFAULT_WORKSPACE_SIZE = 42;
export const MIN_WORKSPACE_SIZE = 24;
export const MAX_WORKSPACE_SIZE = 70;

interface DeskState {
  /** Whether the rail of conversations and places is showing along the left. */
  rail: DeskRailState;
  /** Whether the artifacts pane is showing beside a conversation. */
  pane: DeskPaneState;
  /** The artifact each conversation last had open, so returning to it reopens the same one. */
  activeArtifactByThread: Record<string, string>;
  /** How much of the room the workspace takes, remembered across conversations. */
  workspaceSize: number;
  /** The workspace tab last read. */
  workspaceTab: DeskWorkspaceTab;

  setRail: (rail: DeskRailState) => void;
  toggleRail: () => void;
  setPane: (pane: DeskPaneState) => void;
  togglePane: () => void;
  setActiveArtifact: (threadId: string, artifactId: string | null) => void;
  setWorkspaceSize: (size: number) => void;
  setWorkspaceTab: (tab: DeskWorkspaceTab) => void;
}

/** Whether the rail stands open; anything else is the strip. */
export function railIsOpen(rail: DeskRailState): boolean {
  return rail === "open";
}

/** A workspace share kept inside what the room can give. */
export function clampWorkspaceSize(size: number): number {
  if (!Number.isFinite(size)) {
    return DEFAULT_WORKSPACE_SIZE;
  }

  return Math.min(MAX_WORKSPACE_SIZE, Math.max(MIN_WORKSPACE_SIZE, Math.round(size)));
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
      workspaceSize: DEFAULT_WORKSPACE_SIZE,
      workspaceTab: "artifacts",

      setRail: (rail) => set({ rail }),
      toggleRail: () =>
        set((state) => ({ rail: railIsOpen(state.rail) ? "collapsed" : "open" })),
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
      setWorkspaceSize: (size) => set({ workspaceSize: clampWorkspaceSize(size) }),
      setWorkspaceTab: (tab) => set({ workspaceTab: tab }),
    }),
    {
      name: "trenova-desk",
      partialize: (state) => ({
        rail: state.rail,
        pane: state.pane,
        activeArtifactByThread: state.activeArtifactByThread,
        workspaceSize: state.workspaceSize,
        workspaceTab: state.workspaceTab,
      }),
    },
  ),
);
