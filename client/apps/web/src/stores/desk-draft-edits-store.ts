import { create } from "zustand";

/**
 * The wording a person changed on a draft and sent for approval, keyed by the
 * proposal behind the draft. The draft is a view over a proposal still waiting
 * on a decision: sending it for approval does not decide it, it hands the new
 * wording to the decision card on the composer, which approves with it.
 *
 * Not kept across reloads; an edit sent for approval and never approved goes
 * back to the agent's wording.
 */
interface DraftEditsState {
  edits: Record<string, Record<string, unknown>>;
  /** Keeps the changed parameters for a proposal; an empty set clears them. */
  setEdits: (proposalId: string, modifications: Record<string, unknown>) => void;
  clearEdits: (proposalId: string) => void;
}

export const useDraftEditsStore = create<DraftEditsState>()((set) => ({
  edits: {},
  setEdits: (proposalId, modifications) =>
    set((state) => {
      const next = { ...state.edits };
      if (Object.keys(modifications).length === 0) {
        delete next[proposalId];
      } else {
        next[proposalId] = modifications;
      }
      return { edits: next };
    }),
  clearEdits: (proposalId) =>
    set((state) => {
      if (!(proposalId in state.edits)) {
        return state;
      }
      const next = { ...state.edits };
      delete next[proposalId];
      return { edits: next };
    }),
}));
