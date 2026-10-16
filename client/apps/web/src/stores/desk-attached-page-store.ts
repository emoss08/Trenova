import type { AssistantPageContext } from "@/types/assistant";
import { create } from "zustand";

interface DeskAttachedPageState {
  /** The page a conversation was started about, by conversation. */
  pages: Readonly<Record<string, AssistantPageContext>>;
  attach: (threadId: string, page: AssistantPageContext) => void;
  detach: (threadId: string) => void;
  clear: () => void;
}

/**
 * The page a Desk conversation was started from, when that page is not one
 * the person walked in from: "Ask about this" on a Watchtower item starts a
 * conversation about the item, and every question in it carries the item
 * rather than whatever page was open before the Desk.
 *
 * Kept in memory only, and emptied whenever a session ends: it names
 * business records, which never go to browser storage.
 */
export const useDeskAttachedPageStore = create<DeskAttachedPageState>()((set) => ({
  pages: {},
  attach: (threadId, page) => set((state) => ({ pages: { ...state.pages, [threadId]: page } })),
  detach: (threadId) =>
    set((state) => {
      if (!Object.hasOwn(state.pages, threadId)) {
        return state;
      }
      const { [threadId]: _removed, ...pages } = state.pages;
      return { pages };
    }),
  clear: () => set({ pages: {} }),
}));
