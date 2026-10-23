import {
  RateMatrixRecordDocument,
  RateMatrixVersionAxesDocument,
  type RateMatrixRecordQuery,
  type RateMatrixVersionAxesQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type Signal = { signal?: AbortSignal };

export type RateMatrixRecordVersion = RateMatrixRecordQuery["rateMatrixVersions"][number];
export type RateMatrixRecordReview = RateMatrixRecordQuery["rateMatrixReviews"][number];
export type RateMatrixRecordUsage = RateMatrixRecordQuery["rateMatrixUsages"][number];

/**
 * A rate matrix's draft state with its published versions (newest first), its review rounds
 * (newest first) and the formula template versions that pin one of its versions.
 */
export type RateMatrixRecord = NonNullable<RateMatrixRecordQuery["rateMatrix"]> & {
  versions: RateMatrixRecordVersion[];
  reviews: RateMatrixRecordReview[];
  usages: RateMatrixRecordUsage[];
};

export type RateMatrixVersionAxes = NonNullable<RateMatrixVersionAxesQuery["rateMatrixVersion"]>;

/** Raised when the matrix does not exist or is not the organisation's. */
export class RateMatrixNotFoundError extends Error {
  constructor(rateMatrixId: string) {
    super(`rate matrix ${rateMatrixId} was not found`);
    this.name = "RateMatrixNotFoundError";
  }
}

export async function fetchRateMatrixRecord(
  rateMatrixId: string,
  options?: Signal,
): Promise<RateMatrixRecord> {
  const data = await requestGraphQL({
    document: RateMatrixRecordDocument,
    operationName: "RateMatrixRecord",
    variables: { id: rateMatrixId },
    signal: options?.signal,
  });
  if (!data.rateMatrix) throw new RateMatrixNotFoundError(rateMatrixId);
  return {
    ...data.rateMatrix,
    versions: [...data.rateMatrixVersions].sort((a, b) => b.versionNumber - a.versionNumber),
    reviews: data.rateMatrixReviews,
    usages: data.rateMatrixUsages,
  };
}

/** One published version's axes, or null when the matrix has no such version. */
export async function fetchRateMatrixVersionAxes(
  rateMatrixId: string,
  versionNumber: number,
  options?: Signal,
): Promise<RateMatrixVersionAxes | null> {
  const data = await requestGraphQL({
    document: RateMatrixVersionAxesDocument,
    operationName: "RateMatrixVersionAxes",
    variables: { id: rateMatrixId, versionNumber },
    signal: options?.signal,
  });
  return data.rateMatrixVersion ?? null;
}
