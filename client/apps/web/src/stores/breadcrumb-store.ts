import { create } from "zustand";

interface BreadcrumbState {
  labels: Readonly<Record<string, string>>;
  setLabel: (path: string, label: string) => void;
  clearLabel: (path: string, label: string) => void;
}

/**
 * Labels that pages publish for breadcrumb segments the URL alone cannot
 * name, such as the record an edit page is showing. Keyed by the crumb's
 * full path so a page can name its own crumb or any ancestor's.
 */
export const useBreadcrumbStore = create<BreadcrumbState>()((set) => ({
  labels: {},

  setLabel: (path, label) => {
    set((state) => {
      if (state.labels[path] === label) {
        return state;
      }
      return { labels: { ...state.labels, [path]: label } };
    });
  },

  clearLabel: (path, label) => {
    set((state) => {
      if (state.labels[path] !== label) {
        return state;
      }
      const { [path]: _removed, ...rest } = state.labels;
      return { labels: rest };
    });
  },
}));
