import {
  BulkEditJobFieldsFragmentDoc,
  StartBulkEditDocument,
  UndoMyBulkEditDocument,
  type BulkEditInput,
  type BulkEditJobFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type BulkEditJob = BulkEditJobFieldsFragment;

export const BULK_EDIT_DONE_STATUSES: readonly string[] = ["Completed", "Failed", "Undone"];

export async function startBulkEdit(input: BulkEditInput): Promise<BulkEditJob> {
  const data = await requestGraphQL({
    document: StartBulkEditDocument,
    operationName: "StartBulkEdit",
    variables: { input },
  });
  return getFragmentData(BulkEditJobFieldsFragmentDoc, data.startBulkEdit);
}

export async function undoMyBulkEdit(id: string): Promise<BulkEditJob> {
  const data = await requestGraphQL({
    document: UndoMyBulkEditDocument,
    operationName: "UndoMyBulkEdit",
    variables: { id },
  });
  return getFragmentData(BulkEditJobFieldsFragmentDoc, data.undoMyBulkEdit);
}
