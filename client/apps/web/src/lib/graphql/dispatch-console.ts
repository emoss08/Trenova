import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DispatchAssignMovesDocument,
  DispatchAssignMoveToCarrierDocument,
  DispatchAssignmentPreviewDocument,
  DispatchBoardDocument,
  DispatchBoardMoveFieldsFragmentDoc,
  DispatchCandidateFieldsFragmentDoc,
  DispatchCancelCarrierAssignmentDocument,
  DispatchCarrierAssignmentPreviewDocument,
  DispatchDriverMovesDocument,
  DispatchForecastQualityDocument,
  DispatchForecastQualityFieldsFragmentDoc,
  DispatchFreightOutlookDocument,
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
  type DispatchForecastQualityFieldsFragment,
  type DispatchFreightOutlookQuery,
  type DispatchFreightOutlookQueryVariables,
  type DispatchMoveCandidatesInput,
  type DispatchCandidateFieldsFragment,
  type DispatchPlanAutoAssignMutation,
  type DispatchPlanInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DispatchBoardMove = DispatchBoardMoveFieldsFragment;
export type DispatchMoveProposal = NonNullable<DispatchBoardMove["pendingProposal"]>;
export type DispatchMovePlanPrice = NonNullable<DispatchBoardMove["planPrice"]>;
export type DispatchBoard = Omit<DispatchBoardQuery["dispatchBoard"], "moves"> & {
  moves: DispatchBoardMove[];
};
export type DispatchBoardDriver = DispatchBoard["drivers"][number];
export type DispatchBoardSummary = DispatchBoard["summary"];
export type DispatchCommitment = DispatchBoardDriver["commitments"][number];
export type DispatchTimeOff = DispatchBoardDriver["timeOff"][number];
export type DispatchFinding = DispatchBoardDriver["findings"][number];
export type DispatchDriverPlannedMove = NonNullable<DispatchBoardDriver["plannedMove"]>;
export type DispatchCandidate = DispatchCandidateFieldsFragment;
export type DispatchCostLine = DispatchCandidate["costLines"][number];
export type DispatchAssignmentPreview = Omit<
  DispatchAssignmentPreviewQuery["dispatchAssignmentPreview"],
  "candidate"
> & { candidate: DispatchCandidate };
export type DispatchDriverMoveMatch = Omit<
  DispatchDriverMovesQuery["dispatchDriverMoves"][number],
  "move" | "candidate"
> & { move: DispatchBoardMove; candidate: DispatchCandidate };
export type DispatchBulkAssignResult = DispatchAssignMovesMutation["dispatchAssignMoves"];
export type DispatchCarrierEligibility =
  DispatchCarrierAssignmentPreviewQuery["dispatchCarrierAssignmentPreview"];
export type DispatchCarrierAssignment =
  DispatchAssignMoveToCarrierMutation["dispatchAssignMoveToCarrier"];
type MaskedPlan = DispatchPlanAutoAssignMutation["dispatchPlanAutoAssign"];
export type DispatchPlannedAssignment = Omit<MaskedPlan["assignments"][number], "candidate"> & {
  candidate: DispatchCandidate;
};
export type DispatchUncoveredMove = Omit<MaskedPlan["uncovered"][number], "bestCandidate"> & {
  bestCandidate: DispatchCandidate | null;
};
export type DispatchPlan = Omit<MaskedPlan, "assignments" | "uncovered"> & {
  assignments: DispatchPlannedAssignment[];
  uncovered: DispatchUncoveredMove[];
};
export type DispatchUncoveredReason = DispatchUncoveredMove["reason"];
export type DispatchPlanAlternative = DispatchPlannedAssignment["alternatives"][number];
export type DispatchEndMarket = NonNullable<DispatchCandidate["endMarket"]>;
export type DispatchForecastCustomer = DispatchEndMarket["customers"][number];
export type DispatchForecastQuality = DispatchForecastQualityFieldsFragment;
export type DispatchForecastStatus = DispatchForecastQuality["status"];
export type DispatchFreightOutlook = Omit<
  DispatchFreightOutlookQuery["dispatchFreightOutlook"],
  "quality"
> & { quality: DispatchForecastQuality };
export type DispatchFreightWindow = DispatchFreightOutlook["windows"][number];

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
  return [...getFragmentData(DispatchCandidateFieldsFragmentDoc, data.dispatchMoveCandidates)];
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
    candidate: getFragmentData(DispatchCandidateFieldsFragmentDoc, match.candidate),
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
  const preview = data.dispatchAssignmentPreview;
  return {
    ...preview,
    candidate: getFragmentData(DispatchCandidateFieldsFragmentDoc, preview.candidate),
  };
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
  const plan = data.dispatchPlanAutoAssign;
  return {
    ...plan,
    assignments: plan.assignments.map((assignment) => ({
      ...assignment,
      candidate: getFragmentData(DispatchCandidateFieldsFragmentDoc, assignment.candidate),
    })),
    uncovered: plan.uncovered.map((move) => ({
      ...move,
      bestCandidate:
        getFragmentData(DispatchCandidateFieldsFragmentDoc, move.bestCandidate) ?? null,
    })),
  };
}

/** Freight likely to appear in the hours ahead, by area, with how far the forecast is trusted. */
export async function getDispatchFreightOutlookGraphQL(
  variables: DispatchFreightOutlookQueryVariables,
  options?: { signal?: AbortSignal },
): Promise<DispatchFreightOutlook> {
  const data = await requestGraphQL({
    document: DispatchFreightOutlookDocument,
    operationName: "DispatchFreightOutlook",
    variables,
    signal: options?.signal,
  });
  const outlook = data.dispatchFreightOutlook;
  return {
    ...outlook,
    quality: getFragmentData(DispatchForecastQualityFieldsFragmentDoc, outlook.quality),
  };
}

export async function getDispatchForecastQualityGraphQL(options?: {
  signal?: AbortSignal;
}): Promise<DispatchForecastQuality> {
  const data = await requestGraphQL({
    document: DispatchForecastQualityDocument,
    operationName: "DispatchForecastQuality",
    signal: options?.signal,
  });
  return getFragmentData(DispatchForecastQualityFieldsFragmentDoc, data.dispatchForecastQuality);
}
