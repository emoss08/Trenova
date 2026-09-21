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
  const keys = Object.keys(next);
  if (keys.length <= MAX_DRAFTS) {
    return next;
  }
  for (const key of keys.slice(0, keys.length - MAX_DRAFTS)) {
    delete next[key];
  }

  return next;
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
    }),
    {
      name: "trenova-assistant",
      partialize: (state) => ({
        expanded: state.expanded,
        activeThreadId: state.activeThreadId,
        dismissedSuggestions: state.dismissedSuggestions,
        drafts: state.drafts,
        lastAgentId: state.lastAgentId,
      }),
    },
  ),
);
