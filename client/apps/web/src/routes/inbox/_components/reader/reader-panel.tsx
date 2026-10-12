import type { TriageAction } from "@/lib/graphql/inbox";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { createContext, useContext, useEffect, type Dispatch } from "react";
import type { PanelEvent, PanelState } from "./panel-state";

/**
 * What the open reader shares with its panel, its action bar and the phone's
 * sheet: the panel's state and the moves that change it. It is the reader's
 * own context, so the composer typing or a step opening never reaches the
 * list beside it.
 */
export type ReaderPanel = {
  state: PanelState;
  dispatch: Dispatch<PanelEvent>;
  /** Starts a step the way its kind starts: link, draft, compose or open. */
  start: (action: TriageAction) => void;
  /** Has Nova draft the reply; the composer shows it arriving. */
  draftReply: () => void;
  /** Has Nova draft the load; the form fills in row by row. */
  draftLoad: () => void;
  /**
   * An open step tells the reader what ↵ submits while it is open, and takes
   * it back (null) when it closes or cannot submit.
   */
  registerSubmit: (stepId: string, submit: (() => void) | null) => void;
  /** Opens a file's step, split into its pages when asked. */
  openFile: (attachmentId: string, split: boolean) => void;
  /** The file whose step should open split, once. */
  splitRequested: string | null;
  /** R on a message that asks for no reply opens a composer of its own. */
  adhocReply: boolean;
  /** Opens the reply composer, starting an ad-hoc one when the message has no reply step. */
  openReply: () => void;
  /** The reader is phone-narrow: the panel lives in a sheet. */
  compact: boolean;
};

const ReaderPanelContext = createContext<ReaderPanel | null>(null);

export const ReaderPanelProvider = ReaderPanelContext.Provider;

export function useReaderPanel(): ReaderPanel {
  const panel = useContext(ReaderPanelContext);
  if (panel === null) {
    throw new Error("useReaderPanel is used outside the reader");
  }

  return panel;
}

/**
 * Tells the reader what ↵ submits while this step is open: `submit` when it
 * can, nothing when it cannot (a form with nothing to send, a send in
 * flight). Taken back when the step closes.
 */
export function useStepSubmit(stepId: string, submit: (() => void) | null): void {
  const { registerSubmit } = useReaderPanel();
  const latest = useLatestCallback(() => submit?.());
  const can = submit !== null;

  useEffect(() => {
    registerSubmit(stepId, can ? latest : null);
    return () => registerSubmit(stepId, null);
  }, [registerSubmit, stepId, can, latest]);
}
