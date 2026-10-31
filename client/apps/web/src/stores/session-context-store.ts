import { create } from "zustand";

type SessionContextState = {
  /** The session has moved to another organization than the one this tab loaded. */
  organizationChanged: boolean;
  /**
   * The person chose to keep the page open after being told, to copy what they were working
   * on; the page shows a bar instead of the prompt until it reloads or the session returns.
   */
  keptOpen: boolean;
  setOrganizationChanged: (organizationChanged: boolean) => void;
  keepPageOpen: () => void;
};

export const useSessionContextStore = create<SessionContextState>()((set) => ({
  organizationChanged: false,
  keptOpen: false,
  setOrganizationChanged: (organizationChanged) =>
    set(organizationChanged ? { organizationChanged } : { organizationChanged, keptOpen: false }),
  keepPageOpen: () => set({ keptOpen: true }),
}));
