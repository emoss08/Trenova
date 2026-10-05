import { isPreviewConflict } from "@/components/assistant/proposal-preview/preview-gate";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { commitMyDecisionNow, undoMyDecision } from "@/lib/graphql/agent-decisions";
import { invalidateProposalViews, markPlanStatus, markProposalsStatus } from "@/lib/proposal-cache";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useEffect, useReducer, useRef, useState } from "react";
import { toast } from "sonner";
import { DeskIcon } from "../desk-icons";
import {
  IDLE,
  secondsLeft,
  UNDO_SECONDS,
  undoReducer,
  type UndoState,
  type UndoWindow,
} from "./undo-window";

/** The circumference of the ring's circle (r = 9), which the dash offset counts down. */
const RING = 56.5;
/** How often the countdown looks at the clock; the ring itself moves once a second. */
const TICK_MS = 250;
/** How long the green "Approved" row holds once the change has gone through. */
const COMMITTED_HOLD_MS = 1100;

export type UndoController = {
  state: UndoState;
  /** Seconds left on the ring while the approval can still be undone. */
  left: number;
  start: (window: UndoWindow) => void;
  scheduled: (key: string, commitsAt: number | null) => void;
  clear: (key: string) => void;
  undo: () => void;
  commitNow: () => void;
  busy: boolean;
};

/**
 * The undo window after an approval, as the dock shows it. The server holds
 * the approval and commits it when the window closes; this counts down to
 * that moment, takes it back on Undo, and asks for it now on "Do it now".
 * `onCommitted` runs once the change has gone through.
 */
export function useUndoWindow(
  threadId: string,
  onCommitted: (window: UndoWindow) => void,
): UndoController {
  const t = useT();
  const queryClient = useQueryClient();
  const [state, dispatch] = useReducer(undoReducer, IDLE);
  const [now, setNow] = useState(() => Date.now());

  const waiting = state.phase === "waiting";
  useEffect(() => {
    if (!waiting) {
      return;
    }
    const timer = window.setInterval(() => {
      const at = Date.now();
      setNow(at);
      dispatch({ type: "tick", now: at });
    }, TICK_MS);
    return () => window.clearInterval(timer);
  }, [waiting]);

  const latest = useRef({ onCommitted, state });
  useEffect(() => {
    latest.current = { onCommitted, state };
  });
  const committedKey = state.phase === "committed" ? state.window.key : null;
  useEffect(() => {
    if (committedKey === null) {
      return;
    }
    const current = latest.current.state;
    if (current.phase === "committed") {
      latest.current.onCommitted(current.window);
    }
    const timer = window.setTimeout(
      () => dispatch({ type: "clear", key: committedKey }),
      COMMITTED_HOLD_MS,
    );
    return () => window.clearTimeout(timer);
  }, [committedKey]);

  const undoMutation = useMutation({
    mutationFn: (held: UndoWindow) => undoMyDecision(held.target),
    onSuccess: async (_done, held) => {
      if (held.planId !== null) {
        markPlanStatus(queryClient, held.planId, "Pending");
      } else {
        markProposalsStatus(queryClient, held.proposalIds, "Pending");
      }
      dispatch({ type: "clear", key: held.key });
      await invalidateProposalViews(queryClient, threadId);
    },
    onError: (error, held) => {
      // Too late: the window closed between the click and the server.
      if (isPreviewConflict(error)) {
        toast.info(t("That had already gone through"));
        dispatch({ type: "commit", key: held.key });
        return;
      }
      handleMutationError({ error, resourceName: "Decision" });
    },
  });

  const nowMutation = useMutation({
    mutationFn: (held: UndoWindow) => commitMyDecisionNow(held.target),
    onMutate: (held) => dispatch({ type: "commit", key: held.key }),
    onError: (error) => {
      handleMutationError({ error, resourceName: "Decision" });
      void invalidateProposalViews(queryClient, threadId);
    },
  });

  const start = useCallback((held: UndoWindow) => {
    setNow(held.startedAt);
    dispatch({ type: "start", window: held });
  }, []);
  const scheduled = useCallback(
    (key: string, commitsAt: number | null) => dispatch({ type: "scheduled", key, commitsAt }),
    [],
  );
  const clear = useCallback((key: string) => dispatch({ type: "clear", key }), []);

  const busy = undoMutation.isPending || nowMutation.isPending;
  const windowNow = state.phase === "waiting" ? state.window : null;

  return {
    state,
    left: windowNow ? secondsLeft(windowNow, now) : 0,
    start,
    scheduled,
    clear,
    undo: () => {
      if (windowNow && !busy) {
        undoMutation.mutate(windowNow);
      }
    },
    commitNow: () => {
      if (windowNow && !busy) {
        nowMutation.mutate(windowNow);
      }
    },
    busy,
  };
}

/**
 * The approval in its undo window, attached to the composer where the card
 * was: a ring counting the seconds down, what was approved and what happens
 * when the ring runs out, Undo, and "Do it now".
 */
export function DeskUndoBar({
  held,
  left,
  busy,
  onUndo,
  onNow,
}: {
  held: UndoWindow;
  left: number;
  busy: boolean;
  onUndo: () => void;
  onNow: () => void;
}) {
  const t = useT();
  const title = held.title.charAt(0).toLowerCase() + held.title.slice(1);

  return (
    <div className="dk-udo" role="status" aria-live="polite">
      <svg className="dk-udo-ring" width="22" height="22" viewBox="0 0 22 22" aria-hidden>
        <circle cx="11" cy="11" r="9" className="dk-bg" />
        <circle
          cx="11"
          cy="11"
          r="9"
          className="dk-fg"
          style={{ strokeDashoffset: RING * (1 - left / UNDO_SECONDS) }}
        />
        <text x="11" y="14.5" textAnchor="middle">
          {left}
        </text>
      </svg>
      <span className="dk-udo-t">
        <b>{t("Approved {0}", title)}</b>
        {/* The seconds tick every second; a screen reader is told what will
            happen once, not each tick. */}
        <span aria-hidden>{t("{0} in {1}s", held.what, left)}</span>
        <em className="sr-only">{held.what}</em>
      </span>
      <button type="button" className="dk-bt dk-sm" disabled={busy} onClick={onUndo}>
        <DeskIcon name="undo" size={12} stroke={2} />
        {t("Undo")}
      </button>
      <button type="button" className="dk-bt dk-sm dk-ghost" disabled={busy} onClick={onNow}>
        {t("Do it now")}
      </button>
    </div>
  );
}
