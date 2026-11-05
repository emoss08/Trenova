import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DetentionIntelligenceDocument,
  DetentionIntelligenceTotalsFieldsFragmentDoc,
  DetentionNarrativeSegmentFieldsFragmentDoc,
  DetentionSplitFigureFieldsFragmentDoc,
  type DetentionIntelligenceQuery,
  type DetentionIntelligenceTotalsFieldsFragment,
  type DetentionNarrativeSegmentFieldsFragment,
  type DetentionSplitFigureFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RawIntelligence = DetentionIntelligenceQuery["detentionIntelligence"];
type RawNextStep = RawIntelligence["nextSteps"][number];
type RawSplits = RawIntelligence["splits"];

export type DetentionIntelligenceTotals = DetentionIntelligenceTotalsFieldsFragment;
export type DetentionNarrativeSegment = DetentionNarrativeSegmentFieldsFragment;
export type DetentionSplitFigure = DetentionSplitFigureFieldsFragment;
export type DetentionNarrativeParagraph = { segments: DetentionNarrativeSegment[] };
export type DetentionNextStep = Omit<RawNextStep, "title"> & {
  title: DetentionNarrativeSegment[];
};

/** The read as the page draws it: every masked fragment unmasked once, here. */
export type DetentionIntelligence = Omit<
  RawIntelligence,
  "totals" | "previous" | "splits" | "summary" | "nextSteps"
> & {
  totals: DetentionIntelligenceTotals;
  previous: DetentionIntelligenceTotals;
  splits: Record<keyof RawSplits, DetentionSplitFigure>;
  summary: DetentionNarrativeParagraph[];
  nextSteps: DetentionNextStep[];
};

export type DetentionFacility = DetentionIntelligence["facilities"][number];
export type DetentionCustomer = DetentionIntelligence["customers"][number];
export type DetentionRecoveryStep = DetentionIntelligence["recovery"]["steps"][number];
export type DetentionArrivalCell = DetentionIntelligence["arrivals"][number];

export type DetentionIntelligenceInput = {
  /** How many whole days of the organization's time zone the window covers. */
  days: number;
  /** The window's last day, YYYY-MM-DD in the organization's time zone; today when absent. */
  through?: string;
  /** Narrows every figure to one facility, for its scorecard. */
  locationId?: string;
};

function unmaskSplit(figure: RawSplits["pickup"]): DetentionSplitFigure {
  return getFragmentData(DetentionSplitFigureFieldsFragmentDoc, figure);
}

function unmaskIntelligence(raw: RawIntelligence): DetentionIntelligence {
  return {
    ...raw,
    totals: getFragmentData(DetentionIntelligenceTotalsFieldsFragmentDoc, raw.totals),
    previous: getFragmentData(DetentionIntelligenceTotalsFieldsFragmentDoc, raw.previous),
    splits: {
      pickup: unmaskSplit(raw.splits.pickup),
      delivery: unmaskSplit(raw.splits.delivery),
      appointment: unmaskSplit(raw.splits.appointment),
      open: unmaskSplit(raw.splits.open),
      live: unmaskSplit(raw.splits.live),
      drop: unmaskSplit(raw.splits.drop),
    },
    summary: raw.summary.map((paragraph) => ({
      segments: getFragmentData(DetentionNarrativeSegmentFieldsFragmentDoc, paragraph.segments),
    })),
    nextSteps: raw.nextSteps.map((step) => ({
      ...step,
      title: getFragmentData(DetentionNarrativeSegmentFieldsFragmentDoc, step.title),
    })),
  };
}

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

  return unmaskIntelligence(data.detentionIntelligence);
}
