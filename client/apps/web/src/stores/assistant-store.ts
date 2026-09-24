import {
  DEFAULT_ASSISTANT_DOCK,
  isAssistantDock,
  isPanelSize,
  type AssistantDock,
  type AssistantPanelSize,
} from "@/lib/assistant-dock";
import { create } from "zustand";
import { persist } from "zustand/middleware";

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
export type AssistantOpeningQuestion = {
  threadId: string;
  text: string;
};

interface AssistantState {
  /** Whether the floating panel is showing. */
  open: boolean;
  /** Whether the panel fills the viewport instead of sitting in the corner. */
  expanded: boolean;
  activeThreadId: string | null;
  /** Suggestion prompts the person closed; they stay closed across sessions. */
  dismissedSuggestions: string[];
  /** What was typed but not sent, per conversation, so switching loses nothing. */
  drafts: Record<string, string>;
  /** The agent this person last started a conversation with, on either surface. */
  lastAgentId: string | null;
  /** A question asked before its conversation existed. Never persisted. */
  openingQuestion: AssistantOpeningQuestion | null;
  /** The corner the launcher and the panel sit in. */
  dock: AssistantDock;
  /** The launcher tucked into a tab at the edge of the screen. */
  launcherHidden: boolean;
  /** The panel's size once someone has resized it; the default until then. */
  panelSize: AssistantPanelSize | null;

  openWidget: () => void;
  closeWidget: () => void;
  toggleWidget: () => void;
  setExpanded: (expanded: boolean) => void;
  toggleExpanded: () => void;
  setActiveThreadId: (id: string | null) => void;
  dismissSuggestion: (prompt: string) => void;
  setDraft: (threadId: string, draft: string) => void;
  setLastAgentId: (agentId: string | null) => void;
  setOpeningQuestion: (question: AssistantOpeningQuestion | null) => void;
  setDock: (dock: AssistantDock) => void;
  setLauncherHidden: (hidden: boolean) => void;
  setPanelSize: (size: AssistantPanelSize | null) => void;
}

const MAX_DISMISSED = 50;
/** Drafts kept for the most recently typed-in conversations. */
const MAX_DRAFTS = 20;

/**
 * Stores a draft, or forgets it when emptied, keeping the newest few. The
 * newest is the one just written, so the oldest key is the first inserted.
 */
export function rememberDraft(
  drafts: Record<string, string>,
  threadId: string,
  draft: string,
): Record<string, string> {
  const { [threadId]: _previous, ...rest } = drafts;
  if (draft === "") {
    return rest;
  }
  const next = { ...rest, [threadId]: draft };
  const entries = Object.entries(next);
  if (entries.length <= MAX_DRAFTS) {
    return next;
  }

  return Object.fromEntries(entries.slice(entries.length - MAX_DRAFTS));
}

/**
 * Takes what was saved, but only the parts that still make sense: a corner
 * this build does not know, or a size that is not two numbers, falls back to
 * the default rather than placing the panel somewhere it cannot be seen.
 */
export function mergePersisted(persisted: unknown, current: AssistantState): AssistantState {
  if (typeof persisted !== "object" || persisted === null) {
    return current;
  }
  const saved = persisted as Partial<AssistantState>;

  return {
    ...current,
    ...saved,
    dock: isAssistantDock(saved.dock) ? saved.dock : current.dock,
    launcherHidden: saved.launcherHidden === true,
    panelSize: isPanelSize(saved.panelSize) ? saved.panelSize : null,
  };
}

export const useAssistantStore = create<AssistantState>()(
  persist(
    (set) => ({
      open: false,
      expanded: false,
      activeThreadId: null,
      dismissedSuggestions: [],
      lastAgentId: null,
      openingQuestion: null,
      drafts: {},
      dock: DEFAULT_ASSISTANT_DOCK,
      launcherHidden: false,
      panelSize: null,

      openWidget: () => set({ open: true }),
      closeWidget: () => set({ open: false }),
      toggleWidget: () => set((state) => ({ open: !state.open })),
      setExpanded: (expanded) => set({ expanded }),
      toggleExpanded: () => set((state) => ({ expanded: !state.expanded })),
      setActiveThreadId: (id) => set({ activeThreadId: id }),
      dismissSuggestion: (prompt) =>
        set((state) =>
          state.dismissedSuggestions.includes(prompt)
            ? state
            : {
                dismissedSuggestions: [...state.dismissedSuggestions, prompt].slice(-MAX_DISMISSED),
              },
        ),
      setDraft: (threadId, draft) =>
        set((state) => ({ drafts: rememberDraft(state.drafts, threadId, draft) })),
      setLastAgentId: (agentId) => set({ lastAgentId: agentId }),
      setOpeningQuestion: (question) => set({ openingQuestion: question }),
      setDock: (dock) => set({ dock }),
      setLauncherHidden: (hidden) => set({ launcherHidden: hidden }),
      setPanelSize: (size) => set({ panelSize: size }),
    }),
    {
      name: "trenova-assistant",
      partialize: (state) => ({
        expanded: state.expanded,
        activeThreadId: state.activeThreadId,
        dismissedSuggestions: state.dismissedSuggestions,
        drafts: state.drafts,
        lastAgentId: state.lastAgentId,
        dock: state.dock,
        launcherHidden: state.launcherHidden,
        panelSize: state.panelSize,
      }),
      merge: (persisted, current) => mergePersisted(persisted, current),
    },
  ),
);
