import {
  DEFAULT_ASSISTANT_DOCK,
  DEFAULT_ASSISTANT_LAYOUT,
  isAssistantDock,
  isAssistantLayout,
  isPanelSize,
  type AssistantDock,
  type AssistantLayout,
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

/**
 * A waiting decision the assistant asked the person to make now: one
 * proposal, several of one tool, or a plan. The approval box opens on it
 * whatever else waits, and it is never persisted: a reload starts from the
 * oldest decision again.
 */
export type DecisionFocus = {
  proposalIds: string[];
  planId: string;
};

interface AssistantState {
  /** Whether the panel is showing. */
  open: boolean;
  /**
   * How the panel sits: floating in its corner, docked down one side with the
   * page reflowing beside it, or filling the screen with the conversations
   * listed down its side.
   */
  layout: AssistantLayout;
  /** The conversation list in the full-screen layout folded away (⌘\). */
  sidebarCollapsed: boolean;
  /**
   * A conversation whose reply finished while the panel was closed; the
   * launcher says so until the panel is opened. Never persisted.
   */
  repliedThreadId: string | null;
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
  /**
   * The decisions the person chose to make later, by key ("proposal:…",
   * "plan:…"). The approval box stays a pill for these until reopened; a
   * decision that arrives afterwards opens it again.
   */
  deferredDecisions: string[];
  /** The decision each conversation's approval box was asked to show now. */
  decisionFocus: Record<string, DecisionFocus>;

  openWidget: () => void;
  closeWidget: () => void;
  toggleWidget: () => void;
  setLayout: (layout: AssistantLayout) => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  toggleSidebar: () => void;
  noteReplied: (threadId: string) => void;
  setActiveThreadId: (id: string | null) => void;
  dismissSuggestion: (prompt: string) => void;
  setDraft: (threadId: string, draft: string) => void;
  setLastAgentId: (agentId: string | null) => void;
  setOpeningQuestion: (question: AssistantOpeningQuestion | null) => void;
  setDock: (dock: AssistantDock) => void;
  setLauncherHidden: (hidden: boolean) => void;
  setPanelSize: (size: AssistantPanelSize | null) => void;
  deferDecisions: (keys: readonly string[]) => void;
  resumeDecisions: (keys: readonly string[]) => void;
  focusDecision: (threadId: string, focus: DecisionFocus, keys: readonly string[]) => void;
  clearDecisionFocus: (threadId: string) => void;
}

const MAX_DISMISSED = 50;
/** Drafts kept for the most recently typed-in conversations. */
const MAX_DRAFTS = 20;
/** Deferred decisions remembered; the oldest are forgotten first. */
export const MAX_DEFERRED_DECISIONS = 200;

/** Adds keys to the deferred list, newest last, keeping the newest few. */
export function deferKeys(deferred: readonly string[], keys: readonly string[]): string[] {
  const adding = new Set(keys);
  const kept = deferred.filter((key) => !adding.has(key));

  return [...kept, ...adding].slice(-MAX_DEFERRED_DECISIONS);
}

/** Takes keys off the deferred list. */
export function resumeKeys(deferred: readonly string[], keys: readonly string[]): string[] {
  const removing = new Set(keys);

  return deferred.filter((key) => !removing.has(key));
}

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
 * The layout a saved state asks for. Builds before the three layouts saved
 * only whether the panel was expanded, and expanded is what full screen was.
 */
function savedLayout(saved: Record<string, unknown>): AssistantLayout {
  if (isAssistantLayout(saved.layout)) {
    return saved.layout;
  }
  if (saved.layout === undefined && saved.expanded === true) {
    return "full";
  }

  return DEFAULT_ASSISTANT_LAYOUT;
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
  const { expanded: _expanded, ...saved } = persisted as Partial<AssistantState> & {
    expanded?: unknown;
  };

  return {
    ...current,
    ...saved,
    layout: savedLayout(persisted as Record<string, unknown>),
    sidebarCollapsed: saved.sidebarCollapsed === true,
    repliedThreadId: null,
    dock: isAssistantDock(saved.dock) ? saved.dock : current.dock,
    launcherHidden: saved.launcherHidden === true,
    panelSize: isPanelSize(saved.panelSize) ? saved.panelSize : null,
    deferredDecisions: Array.isArray(saved.deferredDecisions)
      ? saved.deferredDecisions
          .filter((key): key is string => typeof key === "string")
          .slice(-MAX_DEFERRED_DECISIONS)
      : [],
    decisionFocus: {},
  };
}

export const useAssistantStore = create<AssistantState>()(
  persist(
    (set) => ({
      open: false,
      layout: DEFAULT_ASSISTANT_LAYOUT,
      sidebarCollapsed: false,
      repliedThreadId: null,
      activeThreadId: null,
      dismissedSuggestions: [],
      lastAgentId: null,
      openingQuestion: null,
      drafts: {},
      dock: DEFAULT_ASSISTANT_DOCK,
      launcherHidden: false,
      panelSize: null,
      deferredDecisions: [],
      decisionFocus: {},

      openWidget: () => set({ open: true, repliedThreadId: null }),
      closeWidget: () => set({ open: false }),
      toggleWidget: () =>
        set((state) => (state.open ? { open: false } : { open: true, repliedThreadId: null })),
      setLayout: (layout) => set({ layout }),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      noteReplied: (threadId) =>
        set((state) => (state.open ? state : { repliedThreadId: threadId })),
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
      deferDecisions: (keys) =>
        set((state) => ({ deferredDecisions: deferKeys(state.deferredDecisions, keys) })),
      resumeDecisions: (keys) =>
        set((state) => ({ deferredDecisions: resumeKeys(state.deferredDecisions, keys) })),
      focusDecision: (threadId, focus, keys) =>
        set((state) => ({
          decisionFocus: { ...state.decisionFocus, [threadId]: focus },
          deferredDecisions: resumeKeys(state.deferredDecisions, keys),
        })),
      clearDecisionFocus: (threadId) =>
        set((state) => {
          if (!(threadId in state.decisionFocus)) {
            return state;
          }
          const { [threadId]: _cleared, ...rest } = state.decisionFocus;
          return { decisionFocus: rest };
        }),
    }),
    {
      name: "trenova-assistant",
      partialize: (state) => ({
        layout: state.layout,
        sidebarCollapsed: state.sidebarCollapsed,
        activeThreadId: state.activeThreadId,
        dismissedSuggestions: state.dismissedSuggestions,
        drafts: state.drafts,
        lastAgentId: state.lastAgentId,
        dock: state.dock,
        launcherHidden: state.launcherHidden,
        panelSize: state.panelSize,
        deferredDecisions: state.deferredDecisions,
      }),
      merge: (persisted, current) => mergePersisted(persisted, current),
    },
  ),
);
