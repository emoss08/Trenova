import {
  CreateIftaMileageEntryDocument,
  DeleteIftaMileageEntryDocument,
  IftaMileageEntryDocument,
  IftaMileageEntryTableDocument,
  UpdateIftaMileageEntryDocument,
  type CreateIftaMileageEntryMutation,
  type CreateIftaMileageEntryMutationVariables,
  type DeleteIftaMileageEntryMutation,
  type DeleteIftaMileageEntryMutationVariables,
  type IftaMileageEntryFieldsFragment,
  type IftaMileageEntryInput,
  type IftaMileageEntryQuery,
  type IftaMileageEntryQueryVariables,
  type UpdateIftaMileageEntryMutation,
  type UpdateIftaMileageEntryMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type IftaMileageEntryRow = IftaMileageEntryFieldsFragment;

export const IFTA_MILEAGE_ENTRY_LIST_KEY = "ifta-mileage-entry-list";

export const iftaMileageEntryTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: IftaMileageEntryTableDocument,
  operationName: "IftaMileageEntryTable",
  connectionKey: "iftaMileageEntries",
});

export async function fetchIftaMileageEntry(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<IftaMileageEntryRow> {
  const data = await requestGraphQL<IftaMileageEntryQuery, IftaMileageEntryQueryVariables>({
    document: IftaMileageEntryDocument,
    operationName: "IftaMileageEntry",
    variables: { id },
    signal: options?.signal,
  });
  return data.iftaMileageEntry as IftaMileageEntryRow;
}

export async function createIftaMileageEntry(
  input: IftaMileageEntryInput,
): Promise<IftaMileageEntryRow> {
  const data = await requestGraphQL<
    CreateIftaMileageEntryMutation,
    CreateIftaMileageEntryMutationVariables
  >({
    document: CreateIftaMileageEntryDocument,
    operationName: "CreateIftaMileageEntry",
    variables: { input },
  });
  return data.createIftaMileageEntry as IftaMileageEntryRow;
}

export async function updateIftaMileageEntry(
  id: string,
  version: number,
  input: IftaMileageEntryInput,
): Promise<IftaMileageEntryRow> {
  const data = await requestGraphQL<
    UpdateIftaMileageEntryMutation,
    UpdateIftaMileageEntryMutationVariables
  >({
    document: UpdateIftaMileageEntryDocument,
    operationName: "UpdateIftaMileageEntry",
    variables: { id, version, input },
  });
  return data.updateIftaMileageEntry as IftaMileageEntryRow;
}

export async function deleteIftaMileageEntry(id: string, version: number): Promise<boolean> {
  const data = await requestGraphQL<
    DeleteIftaMileageEntryMutation,
    DeleteIftaMileageEntryMutationVariables
  >({
    document: DeleteIftaMileageEntryDocument,
    operationName: "DeleteIftaMileageEntry",
    variables: { id, version },
  });
  return data.deleteIftaMileageEntry;
}
