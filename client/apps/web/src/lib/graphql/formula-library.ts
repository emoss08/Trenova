import {
  FormulaLibraryDocument,
  FormulaLibraryTableDocument,
  type FormulaLibraryQuery,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export const formulaLibraryTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FormulaLibraryTableDocument,
  operationName: "FormulaLibraryTable",
  connectionKey: "formulaLibraries",
});

export type FormulaLibraryRow = DataTableConfigRow<typeof formulaLibraryTableGraphQLConfig>;

/** A library with its versions (newest first) and its review rounds. */
export type FormulaLibraryRecord = NonNullable<FormulaLibraryQuery["formulaLibrary"]>;
export type FormulaLibraryRecordVersion = FormulaLibraryRecord["versions"][number];
export type FormulaLibraryRecordReview = FormulaLibraryRecord["reviews"][number];

/** Raised when the library does not exist or is not the organisation's. */
export class FormulaLibraryNotFoundError extends Error {
  constructor(libraryId: string) {
    super(`formula library ${libraryId} was not found`);
    this.name = "FormulaLibraryNotFoundError";
  }
}

export async function fetchFormulaLibrary(
  libraryId: string,
  options?: { signal?: AbortSignal },
): Promise<FormulaLibraryRecord> {
  const data = await requestGraphQL({
    document: FormulaLibraryDocument,
    operationName: "FormulaLibrary",
    variables: { id: libraryId },
    signal: options?.signal,
  });
  if (!data.formulaLibrary) throw new FormulaLibraryNotFoundError(libraryId);
  return data.formulaLibrary;
}
