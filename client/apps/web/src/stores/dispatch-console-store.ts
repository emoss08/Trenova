import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { createSelectors } from "@trenova/shared/lib/utils";
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
  setDragPreview: (label: string | null) => void;
  openPreflight: (target: PreflightTarget) => void;
  closePreflight: () => void;
  openCarrierAssign: (move: DispatchBoardMove) => void;
  closeCarrierAssign: () => void;
  openCarrierCancel: (move: DispatchBoardMove) => void;
  closeCarrierCancel: () => void;
  reset: () => void;
}

/**
 * Transient interaction state for the console. It is deliberately outside React state so
 * the drag layer, the board panes, and the dialogs can share it without the console
 * re-rendering the whole grid to hand a label down three levels of props. Nothing here is
 * persisted or serialized to the URL — a drag in flight is not a place anyone returns to.
 */
const baseStore = create<DispatchConsoleState>()(
  devtools(
    (set) => ({
      dragPreview: null,
      preflight: null,
      carrierAssignTarget: null,
      carrierCancelTarget: null,

      setDragPreview: (label: string | null) => set({ dragPreview: label }),
      openPreflight: (target: PreflightTarget) => set({ preflight: target, dragPreview: null }),
      closePreflight: () => set({ preflight: null }),
      openCarrierAssign: (move: DispatchBoardMove) => set({ carrierAssignTarget: move }),
      closeCarrierAssign: () => set({ carrierAssignTarget: null }),
      openCarrierCancel: (move: DispatchBoardMove) => set({ carrierCancelTarget: move }),
      closeCarrierCancel: () => set({ carrierCancelTarget: null }),
      reset: () =>
        set({
          dragPreview: null,
          preflight: null,
          carrierAssignTarget: null,
          carrierCancelTarget: null,
        }),
    }),
    { name: "dispatch-console" },
  ),
);

export const useDispatchConsoleStore = createSelectors(baseStore);
