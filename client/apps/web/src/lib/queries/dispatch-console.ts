import {
  getDispatchAssignmentPreviewGraphQL,
  getDispatchBoardGraphQL,
  getDispatchDriverMovesGraphQL,
  getDispatchMoveCandidatesGraphQL,
} from "@/lib/graphql/dispatch-console";
import type {
  DispatchAssignmentPreviewInput,
  DispatchBoardInput,
  DispatchDriverMovesInput,
  DispatchMoveCandidatesInput,
} from "@trenova/graphql/generated/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const dispatchConsole = createQueryKeys("dispatchConsole", {
  board: (input: DispatchBoardInput) => ({
    queryKey: ["dispatch-board", input],
    queryFn: () => getDispatchBoardGraphQL(input),
  }),
  moveCandidates: (input: DispatchMoveCandidatesInput) => ({
    queryKey: ["dispatch-move-candidates", input],
    queryFn: () => getDispatchMoveCandidatesGraphQL(input),
  }),
  driverMoves: (input: DispatchDriverMovesInput) => ({
    queryKey: ["dispatch-driver-moves", input],
    queryFn: () => getDispatchDriverMovesGraphQL(input),
  }),
  assignmentPreview: (input: DispatchAssignmentPreviewInput) => ({
    queryKey: ["dispatch-assignment-preview", input],
    queryFn: () => getDispatchAssignmentPreviewGraphQL(input),
  }),
});
