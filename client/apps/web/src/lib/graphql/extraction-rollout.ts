import {
  ExtractionRolloutAccuracyFieldsFragmentDoc,
  ExtractionRolloutArmFieldsFragmentDoc,
  ExtractionRolloutDocument,
  ExtractionRolloutFieldsFragmentDoc,
  ExtractionRolloutReportDocument,
  UpdateExtractionRolloutDocument,
  type ExtractionRolloutAccuracyFieldsFragment,
  type ExtractionRolloutArmFieldsFragment,
  type ExtractionRolloutFieldsFragment,
  type ExtractionRolloutReportQuery,
  type UpdateExtractionRolloutInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export const EXTRACTION_ROLLOUT_KEY = "extraction-rollout";
export const EXTRACTION_ROLLOUT_REPORT_KEY = "extraction-rollout-report";

export type ExtractionRollout = ExtractionRolloutFieldsFragment;
export type ExtractionRolloutArm = ExtractionRolloutArmFieldsFragment;
export type ExtractionRolloutAccuracy = ExtractionRolloutAccuracyFieldsFragment;
type RawReport = ExtractionRolloutReportQuery["extractionRolloutReport"];

export type ExtractionRolloutReport = Omit<
  RawReport,
  "rollout" | "candidate" | "control" | "candidateAccuracy" | "productionAccuracy"
> & {
  rollout: ExtractionRollout;
  candidate: ExtractionRolloutArm;
  control: ExtractionRolloutArm;
  candidateAccuracy: ExtractionRolloutAccuracy;
  productionAccuracy: ExtractionRolloutAccuracy;
};

export async function fetchExtractionRollout(options?: RequestOptions): Promise<ExtractionRollout> {
  const data = await requestGraphQL({
    document: ExtractionRolloutDocument,
    operationName: "ExtractionRollout",
    variables: {},
    signal: options?.signal,
  });

  return getFragmentData(ExtractionRolloutFieldsFragmentDoc, data.extractionRollout);
}

export async function fetchExtractionRolloutReport(
  options?: RequestOptions,
): Promise<ExtractionRolloutReport> {
  const data = await requestGraphQL({
    document: ExtractionRolloutReportDocument,
    operationName: "ExtractionRolloutReport",
    variables: {},
    signal: options?.signal,
  });
  const report = data.extractionRolloutReport;

  return {
    ...report,
    rollout: getFragmentData(ExtractionRolloutFieldsFragmentDoc, report.rollout),
    candidate: getFragmentData(ExtractionRolloutArmFieldsFragmentDoc, report.candidate),
    control: getFragmentData(ExtractionRolloutArmFieldsFragmentDoc, report.control),
    candidateAccuracy: getFragmentData(
      ExtractionRolloutAccuracyFieldsFragmentDoc,
      report.candidateAccuracy,
    ),
    productionAccuracy: getFragmentData(
      ExtractionRolloutAccuracyFieldsFragmentDoc,
      report.productionAccuracy,
    ),
  };
}

export async function updateExtractionRollout(
  input: UpdateExtractionRolloutInput,
): Promise<ExtractionRollout> {
  const data = await requestGraphQL({
    document: UpdateExtractionRolloutDocument,
    operationName: "UpdateExtractionRollout",
    variables: { input },
  });

  return getFragmentData(ExtractionRolloutFieldsFragmentDoc, data.updateExtractionRollout);
}
