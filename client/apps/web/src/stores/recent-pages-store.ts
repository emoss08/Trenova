import { create } from "zustand";
import { persist } from "zustand/middleware";

export const RECENT_PAGES_LIMIT = 8;

const EMPTY_PAGES: readonly RecentPage[] = [];
const STORAGE_VERSION = 2;

export interface RecentPage {
  path: string;
  title: string;
}

interface RecentPagesState {
  pagesByOrganization: Record<string, RecentPage[]>;
  recordVisit: (organizationId: string | undefined, page: RecentPage) => void;
  forget: (organizationId: string | undefined, path: string) => void;
}

function isRecordable(page: RecentPage): boolean {
  return page.path !== "/" && page.title.trim().length > 0;
}

/**
 * The last few distinct pages a person opened, newest first and kept per
 * organization, so the modules menu can offer a way back to where they just
 * were without pointing at records another organization owns. Home is never
 * recorded: it is one click away everywhere already.
 */
export const useRecentPagesStore = create<RecentPagesState>()(
  persist(
    (set) => ({
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
    }),
    {
      name: "recent-pages-storage",
      version: STORAGE_VERSION,
      partialize: (state) => ({ pagesByOrganization: state.pagesByOrganization }),
      // Version 1 kept one flat list with no organization attached, so there
      // is nothing safe to carry over.
      migrate: () => ({ pagesByOrganization: {} }),
    },
  ),
);

export function useRecentPages(organizationId: string | undefined): readonly RecentPage[] {
  return useRecentPagesStore((state) =>
    organizationId ? (state.pagesByOrganization[organizationId] ?? EMPTY_PAGES) : EMPTY_PAGES,
  );
}
