import {
  DetentionIntelligenceDocument,
  type DetentionIntelligenceQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DetentionIntelligence = DetentionIntelligenceQuery["detentionIntelligence"];
export type DetentionIntelligenceTotals = DetentionIntelligence["totals"];
export type DetentionFacility = DetentionIntelligence["facilities"][number];
export type DetentionCustomer = DetentionIntelligence["customers"][number];
export type DetentionNarrativeParagraph = DetentionIntelligence["summary"][number];
export type DetentionNarrativeSegment = DetentionNarrativeParagraph["segments"][number];
export type DetentionNextStep = DetentionIntelligence["nextSteps"][number];
export type DetentionRecoveryStep = DetentionIntelligence["recovery"]["steps"][number];
export type DetentionArrivalCell = DetentionIntelligence["arrivals"][number];
export type DetentionSplitFigure = DetentionIntelligence["splits"]["pickup"];

export type DetentionIntelligenceInput = {
  /** How many whole days of the organization's time zone the window covers. */
  days: number;
  /** The window's last day, YYYY-MM-DD in the organization's time zone; today when absent. */
  through?: string;
  /** Narrows every figure to one facility, for its scorecard. */
  locationId?: string;
};

export async function fetchDetentionIntelligence(
  input: DetentionIntelligenceInput,
  options?: { signal?: AbortSignal },
): Promise<DetentionIntelligence> {
  const data = await requestGraphQL({
    document: DetentionIntelligenceDocument,
    operationName: "DetentionIntelligence",
    variables: {
      input: {
        days: input.days,
        through: input.through ?? null,
        locationId: input.locationId ?? null,
      },
    },
    signal: options?.signal,
  });

  return data.detentionIntelligence;
}
