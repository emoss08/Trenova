import {
  DeleteIftaTaxRateDocument,
  IftaTaxRateTableDocument,
  UpsertIftaTaxRatesDocument,
  type DeleteIftaTaxRateMutation,
  type DeleteIftaTaxRateMutationVariables,
  type IftaTaxRateFieldsFragment,
  type IftaTaxRateInput,
  type UpsertIftaTaxRatesMutation,
  type UpsertIftaTaxRatesMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type IftaTaxRateRow = IftaTaxRateFieldsFragment;

export const IFTA_TAX_RATE_LIST_KEY = "ifta-tax-rate-list";

export const iftaTaxRateTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: IftaTaxRateTableDocument,
  operationName: "IftaTaxRateTable",
  connectionKey: "iftaTaxRates",
});

export async function upsertIftaTaxRates(input: IftaTaxRateInput[]): Promise<IftaTaxRateRow[]> {
  const data = await requestGraphQL<
    UpsertIftaTaxRatesMutation,
    UpsertIftaTaxRatesMutationVariables
  >({
    document: UpsertIftaTaxRatesDocument,
    operationName: "UpsertIftaTaxRates",
    variables: { input },
  });
  return data.upsertIftaTaxRates as IftaTaxRateRow[];
}

export async function deleteIftaTaxRate(id: string, version: number): Promise<boolean> {
  const data = await requestGraphQL<DeleteIftaTaxRateMutation, DeleteIftaTaxRateMutationVariables>({
    document: DeleteIftaTaxRateDocument,
    operationName: "DeleteIftaTaxRate",
    variables: { id, version },
  });
  return data.deleteIftaTaxRate;
}
