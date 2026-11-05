import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DispatchAssignMovesDocument,
  DispatchAssignMoveToCarrierDocument,
  DispatchAssignmentPreviewDocument,
  DispatchBoardDocument,
  DispatchBoardMoveFieldsFragmentDoc,
  DispatchCancelCarrierAssignmentDocument,
  DispatchCarrierAssignmentPreviewDocument,
  DispatchDriverMovesDocument,
  DispatchMoveCandidatesDocument,
  DispatchMoveDocument,
  DispatchPlanAutoAssignDocument,
  DispatchUnassignMovesDocument,
  type DispatchAssignMoveInput,
  type DispatchAssignMovesMutation,
  type DispatchAssignMoveToCarrierInput,
  type DispatchAssignMoveToCarrierMutation,
  type DispatchAssignmentPreviewInput,
  type DispatchAssignmentPreviewQuery,
  type DispatchBoardInput,
  type DispatchBoardMoveFieldsFragment,
  type DispatchBoardQuery,
  type DispatchCarrierAssignmentPreviewInput,
  type DispatchCarrierAssignmentPreviewQuery,
  type DispatchDriverMovesInput,
  type DispatchDriverMovesQuery,
  type DispatchMoveCandidatesInput,
  type DispatchMoveCandidatesQuery,
  type DispatchPlanAutoAssignMutation,
  type DispatchPlanInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DispatchBoardMove = DispatchBoardMoveFieldsFragment;
export type DispatchMoveProposal = NonNullable<DispatchBoardMove["pendingProposal"]>;
export type DispatchBoard = Omit<DispatchBoardQuery["dispatchBoard"], "moves"> & {
  moves: DispatchBoardMove[];
};
export type DispatchBoardDriver = DispatchBoard["drivers"][number];
export type DispatchBoardSummary = DispatchBoard["summary"];
export type DispatchCommitment = DispatchBoardDriver["commitments"][number];
export type DispatchTimeOff = DispatchBoardDriver["timeOff"][number];
export type DispatchFinding = DispatchBoardDriver["findings"][number];
export type DispatchCandidate = DispatchMoveCandidatesQuery["dispatchMoveCandidates"][number];
export type DispatchScoreFactor = DispatchCandidate["factors"][number];
export type DispatchAssignmentPreview = DispatchAssignmentPreviewQuery["dispatchAssignmentPreview"];
export type DispatchDriverMoveMatch = Omit<
  DispatchDriverMovesQuery["dispatchDriverMoves"][number],
  "move"
> & { move: DispatchBoardMove };
export type DispatchBulkAssignResult = DispatchAssignMovesMutation["dispatchAssignMoves"];
export type DispatchCarrierEligibility =
  DispatchCarrierAssignmentPreviewQuery["dispatchCarrierAssignmentPreview"];
export type DispatchCarrierAssignment =
  DispatchAssignMoveToCarrierMutation["dispatchAssignMoveToCarrier"];
export type DispatchPlan = DispatchPlanAutoAssignMutation["dispatchPlanAutoAssign"];
export type DispatchPlannedAssignment = DispatchPlan["assignments"][number];
export type DispatchUncoveredMove = DispatchPlan["uncovered"][number];

export async function getDispatchBoardGraphQL(
  input: DispatchBoardInput,
  options?: { signal?: AbortSignal },
): Promise<DispatchBoard> {
  const data = await requestGraphQL({
    document: DispatchBoardDocument,
    operationName: "DispatchBoard",
    variables: { input },
    signal: options?.signal,
  });
  const board = data.dispatchBoard;
  return { ...board, moves: [...getFragmentData(DispatchBoardMoveFieldsFragmentDoc, board.moves)] };
}

/** One move as the board draws it, whatever window it falls in. */
export async function getDispatchMoveGraphQL(
  moveId: string,
  options?: { signal?: AbortSignal },
): Promise<DispatchBoardMove> {
  const data = await requestGraphQL({
    document: DispatchMoveDocument,
    operationName: "DispatchMove",
    variables: { moveId },
    signal: options?.signal,
  });
  return getFragmentData(DispatchBoardMoveFieldsFragmentDoc, data.dispatchMove);
}

export async function getDispatchMoveCandidatesGraphQL(
  input: DispatchMoveCandidatesInput,
  options?: { signal?: AbortSignal },
): Promise<DispatchCandidate[]> {
  const data = await requestGraphQL({
    document: DispatchMoveCandidatesDocument,
    operationName: "DispatchMoveCandidates",
    variables: { input },
    signal: options?.signal,
  });
  return data.dispatchMoveCandidates;
}

export async function getDispatchDriverMovesGraphQL(
  input: DispatchDriverMovesInput,
  options?: { signal?: AbortSignal },
): Promise<DispatchDriverMoveMatch[]> {
  const data = await requestGraphQL({
    document: DispatchDriverMovesDocument,
    operationName: "DispatchDriverMoves",
    variables: { input },
    signal: options?.signal,
  });
  return data.dispatchDriverMoves.map((match) => ({
    ...match,
    move: getFragmentData(DispatchBoardMoveFieldsFragmentDoc, match.move),
  }));
}

export async function getDispatchAssignmentPreviewGraphQL(
  input: DispatchAssignmentPreviewInput,
  options?: { signal?: AbortSignal },
): Promise<DispatchAssignmentPreview> {
  const data = await requestGraphQL({
    document: DispatchAssignmentPreviewDocument,
    operationName: "DispatchAssignmentPreview",
    variables: { input },
    signal: options?.signal,
  });
  return data.dispatchAssignmentPreview;
}

export async function assignDispatchMovesGraphQL(
  input: DispatchAssignMoveInput[],
): Promise<DispatchBulkAssignResult> {
  const data = await requestGraphQL({
    document: DispatchAssignMovesDocument,
    operationName: "DispatchAssignMoves",
    variables: { input },
  });
  return data.dispatchAssignMoves;
}

export async function unassignDispatchMovesGraphQL(
  moveIds: string[],
): Promise<DispatchBulkAssignResult> {
  const data = await requestGraphQL({
    document: DispatchUnassignMovesDocument,
    operationName: "DispatchUnassignMoves",
    variables: { moveIds },
  });
  return data.dispatchUnassignMoves;
}

export async function getDispatchCarrierAssignmentPreviewGraphQL(
  input: DispatchCarrierAssignmentPreviewInput,
  options?: { signal?: AbortSignal },
): Promise<DispatchCarrierEligibility> {
  const data = await requestGraphQL({
    document: DispatchCarrierAssignmentPreviewDocument,
    operationName: "DispatchCarrierAssignmentPreview",
    variables: { input },
    signal: options?.signal,
  });
  return data.dispatchCarrierAssignmentPreview;
}

export async function assignDispatchMoveToCarrierGraphQL(
  input: DispatchAssignMoveToCarrierInput,
): Promise<DispatchCarrierAssignment> {
  const data = await requestGraphQL({
    document: DispatchAssignMoveToCarrierDocument,
    operationName: "DispatchAssignMoveToCarrier",
    variables: { input },
  });
  return data.dispatchAssignMoveToCarrier;
}

export async function cancelDispatchCarrierAssignmentGraphQL(
  moveId: string,
  reason: string,
): Promise<boolean> {
  const data = await requestGraphQL({
    document: DispatchCancelCarrierAssignmentDocument,
    operationName: "DispatchCancelCarrierAssignment",
    variables: { moveId, reason },
  });
  return data.dispatchCancelCarrierAssignment;
}

export async function planDispatchAutoAssignGraphQL(
  input: DispatchPlanInput,
): Promise<DispatchPlan> {
  const data = await requestGraphQL({
    document: DispatchPlanAutoAssignDocument,
    operationName: "DispatchPlanAutoAssign",
    variables: { input },
  });
  return data.dispatchPlanAutoAssign;
}
