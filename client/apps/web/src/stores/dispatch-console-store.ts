import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { create } from "zustand";
import { devtools } from "zustand/middleware";

/** The pairing a drop proposes, held until the dispatcher confirms or cancels it. */
export type PreflightTarget = {
  move: DispatchBoardMove;
  driver: DispatchBoardDriver;
};

interface DispatchConsoleState {
  /** Label shown under the cursor while a driver or move is being dragged. */
  dragPreview: string | null;
  preflight: PreflightTarget | null;
  /** Move the dispatcher is brokering out to an external carrier. */
  carrierAssignTarget: DispatchBoardMove | null;
  /** Carrier-covered move whose coverage the dispatcher is canceling. */
  carrierCancelTarget: DispatchBoardMove | null;
  /** Move the dispatcher is tendering out through a waterfall or spot offers. */
  tenderTarget: DispatchBoardMove | null;
  /**
   * Whether the inspector's ranked drivers are open. It holds from move to move, so a
   * dispatcher who folds them away to read proposals keeps them folded while walking the queue.
   */
  candidatesOpen: boolean;
  /** Whether the inspector's tender history is open; it holds from move to move the same way. */
  tendersOpen: boolean;
  setDragPreview: (label: string | null) => void;
  openPreflight: (target: PreflightTarget) => void;
  closePreflight: () => void;
  openCarrierAssign: (move: DispatchBoardMove) => void;
  closeCarrierAssign: () => void;
  openCarrierCancel: (move: DispatchBoardMove) => void;
  closeCarrierCancel: () => void;
  openTender: (move: DispatchBoardMove) => void;
  closeTender: () => void;
  setCandidatesOpen: (open: boolean) => void;
  setTendersOpen: (open: boolean) => void;
  reset: () => void;
}

/**
 * Transient interaction state for the console. It is deliberately outside React state so
 * the drag layer, the board panes, and the dialogs can share it without the console
 * re-rendering the whole grid to hand a label down three levels of props. Nothing here is
 * persisted or serialized to the URL — a drag in flight is not a place anyone returns to.
 */
export const useDispatchConsoleStore = create<DispatchConsoleState>()(
  devtools(
    (set) => ({
      dragPreview: null,
      preflight: null,
      carrierAssignTarget: null,
      carrierCancelTarget: null,
      tenderTarget: null,
      candidatesOpen: true,
      tendersOpen: true,

      setDragPreview: (label: string | null) => set({ dragPreview: label }),
      openPreflight: (target: PreflightTarget) => set({ preflight: target, dragPreview: null }),
      closePreflight: () => set({ preflight: null }),
      openCarrierAssign: (move: DispatchBoardMove) => set({ carrierAssignTarget: move }),
      closeCarrierAssign: () => set({ carrierAssignTarget: null }),
      openCarrierCancel: (move: DispatchBoardMove) => set({ carrierCancelTarget: move }),
      closeCarrierCancel: () => set({ carrierCancelTarget: null }),
      openTender: (move: DispatchBoardMove) => set({ tenderTarget: move }),
      closeTender: () => set({ tenderTarget: null }),
      setCandidatesOpen: (open: boolean) => set({ candidatesOpen: open }),
      setTendersOpen: (open: boolean) => set({ tendersOpen: open }),
      reset: () =>
        set({
          dragPreview: null,
          preflight: null,
          carrierAssignTarget: null,
          carrierCancelTarget: null,
          tenderTarget: null,
          candidatesOpen: true,
          tendersOpen: true,
        }),
    }),
    { name: "dispatch-console" },
  ),
);
