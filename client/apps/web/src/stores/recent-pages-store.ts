import { create } from "zustand";

export const RECENT_PAGES_LIMIT = 8;

const EMPTY_PAGES: readonly RecentPage[] = [];

export interface RecentPage {
  path: string;
  title: string;
}

interface RecentPagesState {
  pagesByOrganization: Record<string, RecentPage[]>;
  recordVisit: (organizationId: string | undefined, page: RecentPage) => void;
  forget: (organizationId: string | undefined, path: string) => void;
  /** Forgets every page; the session that visited them has ended. */
  clear: () => void;
}

function isRecordable(page: RecentPage): boolean {
  return page.path !== "/" && page.title.trim().length > 0;
}

/**
 * The last few distinct pages a person opened, newest first and kept per
 * organization, so the modules menu can offer a way back to where they just
 * were without pointing at records another organization owns. Home is never
 * recorded: it is one click away everywhere already.
 *
 * Kept in memory only, never in browser storage: a record page is remembered by the
 * record's name, which a shared dispatch-office PC would keep for the next person.
 * clearSessionCache forgets it all when the session ends or changes organization.
 */
export const useRecentPagesStore = create<RecentPagesState>()((set) => ({
  pagesByOrganization: {},

  recordVisit: (organizationId, page) => {
    if (!organizationId || !isRecordable(page)) {
      return;
    }
    set((state) => {
      const current = state.pagesByOrganization[organizationId] ?? EMPTY_PAGES;
      const others = current.filter((entry) => entry.path !== page.path);
      const next = [{ path: page.path, title: page.title.trim() }, ...others].slice(
        0,
        RECENT_PAGES_LIMIT,
      );
      return {
        pagesByOrganization: { ...state.pagesByOrganization, [organizationId]: next },
      };
    });
  },

  forget: (organizationId, path) => {
    if (!organizationId) {
      return;
    }
    set((state) => {
      const current = state.pagesByOrganization[organizationId];
      if (!current) {
        return state;
      }
      return {
        pagesByOrganization: {
          ...state.pagesByOrganization,
          [organizationId]: current.filter((entry) => entry.path !== path),
        },
      };
    });
  },

  clear: () => set({ pagesByOrganization: {} }),
}));

export function useRecentPages(organizationId: string | undefined): readonly RecentPage[] {
  return useRecentPagesStore((state) =>
    organizationId ? (state.pagesByOrganization[organizationId] ?? EMPTY_PAGES) : EMPTY_PAGES,
  );
}
