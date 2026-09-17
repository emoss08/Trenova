import { create } from "zustand";
import { persist } from "zustand/middleware";

interface AssistantState {
  /** Whether the floating panel is showing. */
  open: boolean;
  /** Whether the panel fills the viewport instead of sitting in the corner. */
  expanded: boolean;
  activeThreadId: string | null;
  /** Suggestion prompts the person closed; they stay closed across sessions. */
  dismissedSuggestions: string[];

  openWidget: () => void;
  closeWidget: () => void;
  toggleWidget: () => void;
  setExpanded: (expanded: boolean) => void;
  toggleExpanded: () => void;
  setActiveThreadId: (id: string | null) => void;
  dismissSuggestion: (prompt: string) => void;
}

const MAX_DISMISSED = 50;

export const useAssistantStore = create<AssistantState>()(
  persist(
    (set) => ({
      open: false,
      expanded: false,
      activeThreadId: null,
      dismissedSuggestions: [],

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
    }),
    {
      name: "trenova-assistant",
      partialize: (state) => ({
        expanded: state.expanded,
        activeThreadId: state.activeThreadId,
        dismissedSuggestions: state.dismissedSuggestions,
      }),
    },
  ),
);
