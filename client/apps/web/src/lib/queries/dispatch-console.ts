import {
  getDispatchAssignmentPreviewGraphQL,
  getDispatchBoardGraphQL,
  getDispatchCarrierAssignmentPreviewGraphQL,
  getDispatchDriverMovesGraphQL,
  getDispatchMoveCandidatesGraphQL,
} from "@/lib/graphql/dispatch-console";
import type {
  DispatchAssignmentPreviewInput,
  DispatchBoardInput,
  DispatchCarrierAssignmentPreviewInput,
  DispatchDriverMovesInput,
  DispatchMoveCandidatesInput,
} from "@trenova/graphql/generated/graphql";

/**
 * Dispatch console queries use flat string-prefixed keys instead of the merged
 * query-key factory: the CDC realtime-patching map invalidates the "dispatch-board"
 * prefix directly, and post-mutation invalidation relies on the same prefixes.
 */
export const DISPATCH_BOARD_KEY = "dispatch-board";
export const DISPATCH_MOVE_CANDIDATES_KEY = "dispatch-move-candidates";
export const DISPATCH_DRIVER_MOVES_KEY = "dispatch-driver-moves";
export const DISPATCH_ASSIGNMENT_PREVIEW_KEY = "dispatch-assignment-preview";
export const DISPATCH_CARRIER_PREVIEW_KEY = "dispatch-carrier-assignment-preview";

export const dispatchConsoleQueries = {
  board: (input: DispatchBoardInput) => ({
    queryKey: [DISPATCH_BOARD_KEY, input] as const,
    queryFn: () => getDispatchBoardGraphQL(input),
  }),
  moveCandidates: (input: DispatchMoveCandidatesInput) => ({
    queryKey: [DISPATCH_MOVE_CANDIDATES_KEY, input] as const,
    queryFn: () => getDispatchMoveCandidatesGraphQL(input),
  }),
  driverMoves: (input: DispatchDriverMovesInput) => ({
    queryKey: [DISPATCH_DRIVER_MOVES_KEY, input] as const,
    queryFn: () => getDispatchDriverMovesGraphQL(input),
  }),
  assignmentPreview: (input: DispatchAssignmentPreviewInput) => ({
    queryKey: [DISPATCH_ASSIGNMENT_PREVIEW_KEY, input] as const,
    queryFn: () => getDispatchAssignmentPreviewGraphQL(input),
  }),
  carrierAssignmentPreview: (input: DispatchCarrierAssignmentPreviewInput) => ({
    queryKey: [DISPATCH_CARRIER_PREVIEW_KEY, input] as const,
    queryFn: () => getDispatchCarrierAssignmentPreviewGraphQL(input),
  }),
};
