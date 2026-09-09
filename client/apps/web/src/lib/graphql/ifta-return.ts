import {
  AmendIftaReturnDocument,
  BackfillJurisdictionMilesDocument,
  DeleteIftaReturnDocument,
  FinalizeIftaReturnDocument,
  GenerateIftaReturnDocument,
  IftaCurrentPeriodDocument,
  IftaPeriodDocument,
  IftaReturnDocument,
  IftaReturnForPeriodDocument,
  IftaReturnTableDocument,
  MarkIftaReturnFiledDocument,
  RecomputeIftaReturnDocument,
  ReopenIftaReturnDocument,
  type AmendIftaReturnMutation,
  type AmendIftaReturnMutationVariables,
  type BackfillJurisdictionMilesInput,
  type BackfillJurisdictionMilesMutation,
  type BackfillJurisdictionMilesMutationVariables,
  type DeleteIftaReturnMutation,
  type DeleteIftaReturnMutationVariables,
  type FinalizeIftaReturnMutation,
  type FinalizeIftaReturnMutationVariables,
  type GenerateIftaReturnMutation,
  type GenerateIftaReturnMutationVariables,
  type IftaCurrentPeriodQuery,
  type IftaCurrentPeriodQueryVariables,
  type IftaPeriodFieldsFragment,
  type IftaPeriodInput,
  type IftaPeriodQuery,
  type IftaPeriodQueryVariables,
  type IftaReturnFieldsFragment,
  type IftaReturnForPeriodQuery,
  type IftaReturnForPeriodQueryVariables,
  type IftaReturnLineFieldsFragment,
  type IftaReturnQuery,
  type IftaReturnQueryVariables,
  type IftaReturnSummaryFieldsFragment,
  type MarkIftaReturnFiledInput,
  type MarkIftaReturnFiledMutation,
  type MarkIftaReturnFiledMutationVariables,
  type RecomputeIftaReturnMutation,
  type RecomputeIftaReturnMutationVariables,
  type ReopenIftaReturnMutation,
  type ReopenIftaReturnMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type IftaReturn = IftaReturnFieldsFragment;
export type IftaReturnLine = IftaReturnLineFieldsFragment;
export type IftaReturnSummaryRow = IftaReturnSummaryFieldsFragment;
export type IftaPeriod = IftaPeriodFieldsFragment;
export type JurisdictionMilesBackfillResult =
  BackfillJurisdictionMilesMutation["backfillJurisdictionMiles"];

export const IFTA_RETURN_KEY = "ifta-return";
export const IFTA_RETURN_LIST_KEY = "ifta-return-list";
export const IFTA_PERIOD_KEY = "ifta-period";

type RequestOptions = { signal?: AbortSignal };

export const iftaReturnTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: IftaReturnTableDocument,
  operationName: "IftaReturnTable",
  connectionKey: "iftaReturns",
});

export async function fetchIftaReturnForPeriod(
  input: IftaPeriodInput,
  options?: RequestOptions,
): Promise<IftaReturn | null> {
  const data = await requestGraphQL<IftaReturnForPeriodQuery, IftaReturnForPeriodQueryVariables>(
    {
      document: IftaReturnForPeriodDocument,
      operationName: "IftaReturnForPeriod",
      variables: { input },
      signal: options?.signal,
    },
  );
  return (data.iftaReturnForPeriod as IftaReturn | null | undefined) ?? null;
}

export async function fetchIftaReturn(id: string, options?: RequestOptions): Promise<IftaReturn> {
  const data = await requestGraphQL<IftaReturnQuery, IftaReturnQueryVariables>({
    document: IftaReturnDocument,
    operationName: "IftaReturn",
    variables: { id },
    signal: options?.signal,
  });
  return data.iftaReturn as IftaReturn;
}

export async function fetchIftaCurrentPeriod(options?: RequestOptions): Promise<IftaPeriod> {
  const data = await requestGraphQL<IftaCurrentPeriodQuery, IftaCurrentPeriodQueryVariables>({
    document: IftaCurrentPeriodDocument,
    operationName: "IftaCurrentPeriod",
    variables: {},
    signal: options?.signal,
  });
  return data.iftaCurrentPeriod as IftaPeriod;
}

export async function fetchIftaPeriod(
  input: IftaPeriodInput,
  options?: RequestOptions,
): Promise<IftaPeriod> {
  const data = await requestGraphQL<IftaPeriodQuery, IftaPeriodQueryVariables>({
    document: IftaPeriodDocument,
    operationName: "IftaPeriod",
    variables: { year: input.year, quarter: input.quarter },
    signal: options?.signal,
  });
  return data.iftaPeriod as IftaPeriod;
}

export async function generateIftaReturn(period: IftaPeriodInput): Promise<IftaReturn> {
  const data = await requestGraphQL<GenerateIftaReturnMutation, GenerateIftaReturnMutationVariables>(
    {
      document: GenerateIftaReturnDocument,
      operationName: "GenerateIftaReturn",
      variables: { period },
    },
  );
  return data.generateIftaReturn as IftaReturn;
}

export async function recomputeIftaReturn(id: string, version: number): Promise<IftaReturn> {
  const data = await requestGraphQL<
    RecomputeIftaReturnMutation,
    RecomputeIftaReturnMutationVariables
  >({
    document: RecomputeIftaReturnDocument,
    operationName: "RecomputeIftaReturn",
    variables: { id, version },
  });
  return data.recomputeIftaReturn as IftaReturn;
}

export async function finalizeIftaReturn(id: string, version: number): Promise<IftaReturn> {
  const data = await requestGraphQL<FinalizeIftaReturnMutation, FinalizeIftaReturnMutationVariables>(
    {
      document: FinalizeIftaReturnDocument,
      operationName: "FinalizeIftaReturn",
      variables: { id, version },
    },
  );
  return data.finalizeIftaReturn as IftaReturn;
}

export async function reopenIftaReturn(params: {
  id: string;
  version: number;
  reason: string;
}): Promise<IftaReturn> {
  const data = await requestGraphQL<ReopenIftaReturnMutation, ReopenIftaReturnMutationVariables>({
    document: ReopenIftaReturnDocument,
    operationName: "ReopenIftaReturn",
    variables: params,
  });
  return data.reopenIftaReturn as IftaReturn;
}

export async function markIftaReturnFiled(input: MarkIftaReturnFiledInput): Promise<IftaReturn> {
  const data = await requestGraphQL<
    MarkIftaReturnFiledMutation,
    MarkIftaReturnFiledMutationVariables
  >({
    document: MarkIftaReturnFiledDocument,
    operationName: "MarkIftaReturnFiled",
    variables: { input },
  });
  return data.markIftaReturnFiled as IftaReturn;
}

export async function amendIftaReturn(params: { id: string; reason: string }): Promise<IftaReturn> {
  const data = await requestGraphQL<AmendIftaReturnMutation, AmendIftaReturnMutationVariables>({
    document: AmendIftaReturnDocument,
    operationName: "AmendIftaReturn",
    variables: params,
  });
  return data.amendIftaReturn as IftaReturn;
}

export async function deleteIftaReturn(id: string, version: number): Promise<boolean> {
  const data = await requestGraphQL<DeleteIftaReturnMutation, DeleteIftaReturnMutationVariables>({
    document: DeleteIftaReturnDocument,
    operationName: "DeleteIftaReturn",
    variables: { id, version },
  });
  return data.deleteIftaReturn;
}

export async function backfillJurisdictionMiles(
  input: BackfillJurisdictionMilesInput,
): Promise<JurisdictionMilesBackfillResult> {
  const data = await requestGraphQL<
    BackfillJurisdictionMilesMutation,
    BackfillJurisdictionMilesMutationVariables
  >({
    document: BackfillJurisdictionMilesDocument,
    operationName: "BackfillJurisdictionMiles",
    variables: { input },
  });
  return data.backfillJurisdictionMiles;
}
