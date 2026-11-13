import { create } from "zustand";
import { installTourConsole, readTourAlways } from "./page-tour-console";

interface PageTourState {
  /** The tours a page on screen can run right now, by key. */
  available: readonly string[];
  /** The tour somebody asked to see again, until the page that owns it closes it. */
  requested: string | null;
  /** The tour open right now, so its page can draw every part the tour points at. */
  showing: string | null;
  /**
   * A developer's switch (`tour.always()` in the console): tours open on every visit, seen
   * or not. Never on in a release.
   */
  always: boolean;
  announce: (tour: string) => void;
  withdraw: (tour: string) => void;
  request: (tour: string) => void;
  clearRequest: () => void;
  show: (tour: string) => void;
  hide: (tour: string) => void;
  setAlways: (on: boolean) => void;
}

/**
 * How the page header asks a page to replay its tour. The tour lives in the page, where its
 * steps and their targets are; the "Take the tour" button lives in the header's About note.
 * Nothing here is kept: which tours a person has seen is the server's to remember.
 */
export const usePageTourStore = create<PageTourState>()((set) => ({
  available: [],
  requested: null,
  showing: null,
  always: import.meta.env.DEV && readTourAlways(),
  announce: (tour) =>
    set((state) =>
      state.available.includes(tour) ? state : { available: [...state.available, tour] },
    ),
  withdraw: (tour) =>
    set((state) => ({
      available: state.available.filter((key) => key !== tour),
      requested: state.requested === tour ? null : state.requested,
    })),
  request: (tour) => set({ requested: tour }),
  clearRequest: () => set({ requested: null }),
  show: (tour) => set({ showing: tour }),
  hide: (tour) => set((state) => (state.showing === tour ? { showing: null } : state)),
  setAlways: (on) => set({ always: on }),
}));

if (import.meta.env.DEV) installTourConsole(usePageTourStore);
