import { create } from "zustand";
import type { DeskSubject } from "./desk-margin-store";

interface DeskAskState {
  /** The record the next conversation asked for on Today's box is about, if any. */
  subject: DeskSubject | null;
  /** A new conversation was asked for: Today's box takes the caret once it can. */
  focusPending: boolean;
  askNew: (subject: DeskSubject | null) => void;
  /** Takes the caret request, once; false when there was none. */
  takeFocus: () => boolean;
  reset: () => void;
}

/**
 * A new conversation asked for from anywhere on the Desk. Every new
 * conversation starts full width in Today's box; this carries the record the
 * page was about and the caret request across the navigation there.
 */
export const useDeskAskStore = create<DeskAskState>()((set, get) => ({
  subject: null,
  focusPending: false,

  askNew: (subject) => set({ subject, focusPending: true }),
  takeFocus: () => {
    if (!get().focusPending) {
      return false;
    }
    set({ focusPending: false });
    return true;
  },
  reset: () => {
    if (get().subject !== null || get().focusPending) {
      set({ subject: null, focusPending: false });
    }
  },
}));
