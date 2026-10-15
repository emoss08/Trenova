import {
  DeskDetentionCardDocument,
  type DeskDetentionCardQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import {
  calculationTraceSchema,
  type CalculationTrace,
  policySnapshotSchema,
  type PolicySnapshot,
} from "@trenova/shared/types/detention";

/** The card's query answer as the server sends it. */
export type CardDetail = DeskDetentionCardQuery["detentionOccurrence"];
type CardOccurrence = CardDetail["occurrence"];

/**
 * The occurrence as the card reads it: the server's record with its policy
 * snapshot and calculation trace parsed, each null when the record carries
 * none or one the client cannot read.
 */
export type DeskDetentionOccurrence = Omit<
  CardOccurrence,
  "policySnapshot" | "calculationTrace"
> & {
  snapshot: PolicySnapshot | null;
  trace: CalculationTrace | null;
};

export type DeskDetention = Omit<CardDetail, "occurrence" | "evidence" | "notices"> & {
  occurrence: DeskDetentionOccurrence;
  evidence: NonNullable<CardDetail["evidence"]>;
  notices: NonNullable<CardDetail["notices"]>;
};
export type DeskDetentionEvidence = DeskDetention["evidence"][number];
export type DeskDetentionNotice = DeskDetention["notices"][number];
export type DeskDetentionFactor = DeskDetention["collectability"]["factors"][number];
export type DeskDetentionPrice = NonNullable<DeskDetention["currentlyBillable"]>;

/**
 * One stop's detention charge as the Desk's card reads it: who signed it off,
 * its location's place, a running clock priced as of now, and the evidence,
 * notices and collectability the claim rests on.
 */
export async function fetchDeskDetention(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskDetention> {
  const data = await requestGraphQL({
    document: DeskDetentionCardDocument,
    operationName: "DeskDetentionCard",
    variables: { id },
    signal: options?.signal,
  });
  return toDeskDetention(data.detentionOccurrence);
}

/**
 * The server's answer as the card reads it: the policy snapshot and the
 * calculation trace parsed, each null when the record carries none or one
 * the client cannot read, and evidence and notices as lists, empty when the
 * stop has none.
 */
export function toDeskDetention(detail: CardDetail): DeskDetention {
  const { policySnapshot, calculationTrace, ...occurrence } = detail.occurrence;
  const snapshot = policySnapshotSchema.safeParse(policySnapshot);
  const trace = calculationTraceSchema.safeParse(calculationTrace);
  return {
    ...detail,
    occurrence: {
      ...occurrence,
      snapshot: snapshot.success ? snapshot.data : null,
      trace: trace.success ? trace.data : null,
    },
    evidence: detail.evidence ?? [],
    notices: detail.notices ?? [],
  };
}
