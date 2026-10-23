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

export type FormulaLibraryRecordVersion = FormulaLibraryQuery["formulaLibraryVersions"][number];
export type FormulaLibraryRecordReview = FormulaLibraryQuery["formulaLibraryReviews"][number];

/**
 * A library with its versions (newest first) and its review rounds (newest first), read in one
 * request, and the approved version new template versions pin.
 */
export type FormulaLibraryRecord = NonNullable<FormulaLibraryQuery["formulaLibrary"]> & {
  versions: FormulaLibraryRecordVersion[];
  reviews: FormulaLibraryRecordReview[];
  /** The newest Active version, or null when none was ever approved. */
  approvedVersionNumber: number | null;
};

/** Raised when the library does not exist or is not the organisation's. */
export class FormulaLibraryNotFoundError extends Error {
  constructor(libraryId: string) {
    super(`formula library ${libraryId} was not found`);
    this.name = "FormulaLibraryNotFoundError";
  }
}

/** The newest approved version among a library's versions. */
export function approvedVersionOf(versions: readonly FormulaLibraryRecordVersion[]): number | null {
  let approved: number | null = null;
  for (const version of versions) {
    if (version.status === "Active" && (approved === null || version.versionNumber > approved)) {
      approved = version.versionNumber;
    }
  }
  return approved;
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
  const versions = [...data.formulaLibraryVersions].sort(
    (a, b) => b.versionNumber - a.versionNumber,
  );
  return {
    ...data.formulaLibrary,
    versions,
    reviews: data.formulaLibraryReviews,
    approvedVersionNumber: approvedVersionOf(versions),
  };
}
