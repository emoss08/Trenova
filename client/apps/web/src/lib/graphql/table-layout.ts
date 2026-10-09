import {
  ResetMyTableLayoutDocument,
  SaveMyTableLayoutDocument,
  type MyTableLayoutQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { TableLayout } from "@/types/table-configuration";

export type SavedTableLayout = NonNullable<MyTableLayoutQuery["myTableLayout"]>;

export async function saveMyTableLayout(
  resource: string,
  layout: TableLayout,
): Promise<SavedTableLayout> {
  const data = await requestGraphQL({
    document: SaveMyTableLayoutDocument,
    operationName: "SaveMyTableLayout",
    variables: { input: { resource, layout } },
  });

  return data.saveMyTableLayout;
}

export async function resetMyTableLayout(resource: string): Promise<boolean> {
  const data = await requestGraphQL({
    document: ResetMyTableLayoutDocument,
    operationName: "ResetMyTableLayout",
    variables: { resource },
  });

  return data.resetMyTableLayout;
}
