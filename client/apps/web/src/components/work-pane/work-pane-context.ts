import type { PanelCloseGuard } from "@/components/form-changes";
import { createContext, useContext, type ReactNode } from "react";

/** What a pane hands its page when it claims the host. */
export type WorkPaneEntry = {
  /** Closes the pane, asking first when its form holds changes, then runs `then`. */
  requestClose: (then?: () => void) => void;
  /** Another pane took the host: close without asking. */
  displace: () => void;
};

export type WorkPaneApi = {
  /** Claims the host for pane `id`, closing any other; returns the release. */
  register: (id: string, entry: WorkPaneEntry) => () => void;
  setDirty: (id: string, dirty: boolean) => void;
  closeThen: (then: () => void) => void;
  close: () => void;
  setHost: (element: HTMLDivElement | null) => void;
};

export type WorkPaneState = {
  /** The pane showing in the host, or null with the host closed. */
  activeId: string | null;
  /** The pane showing holds changes not saved. */
  isDirty: boolean;
  host: HTMLDivElement | null;
};

export const WorkPaneApiContext = createContext<WorkPaneApi | null>(null);
export const WorkPaneStateContext = createContext<WorkPaneState | null>(null);

/** What the parts inside one pane share. */
export type WorkPanePart = {
  guard: PanelCloseGuard;
  requestClose: () => void;
  description: ReactNode;
  descriptionId: string;
  registerFooter: () => () => void;
};

export const WorkPanePartContext = createContext<WorkPanePart | null>(null);

export function useWorkPaneApi(): WorkPaneApi {
  const api = useContext(WorkPaneApiContext);
  if (!api) throw new Error("A work pane needs a WorkPaneProvider above it.");
  return api;
}

export function useWorkPaneState(): WorkPaneState {
  const state = useContext(WorkPaneStateContext);
  if (!state) throw new Error("A work pane needs a WorkPaneProvider above it.");
  return state;
}

export function useWorkPanePart(): WorkPanePart {
  const part = useContext(WorkPanePartContext);
  if (!part) throw new Error("A work pane part must sit inside a WorkPane.");
  return part;
}

export type WorkPaneControls = {
  /** A pane is showing. */
  isOpen: boolean;
  /** The pane showing holds changes not saved: OR it into the page's `UnsavedChangesGuard`. */
  isDirty: boolean;
  /**
   * Closes the pane showing, asking first in its footer when its form holds changes, then runs
   * `then`; with no pane showing, runs `then` at once. Wrap every tab, view or pane change in it.
   */
  closeThen: (then: () => void) => void;
  /** Closes the pane showing, asking first when its form holds changes. */
  close: () => void;
};

/** The page's handle on its work pane, for anything under its `WorkPaneProvider`. */
export function useWorkPane(): WorkPaneControls {
  const api = useWorkPaneApi();
  const state = useWorkPaneState();
  return {
    isOpen: state.activeId !== null,
    isDirty: state.isDirty,
    closeThen: api.closeThen,
    close: api.close,
  };
}
