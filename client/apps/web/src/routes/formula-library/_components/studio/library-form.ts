// library-form.ts holds what the library pages' form starts from and how it relates to the
// saved library.

import type { FormulaLibraryRecord } from "@/lib/graphql/formula-library";
import type {
  FormulaLibrary,
  FormulaLibraryFormValues,
} from "@trenova/shared/types/formula-library";

export const EMPTY_LIBRARY: FormulaLibraryFormValues = {
  name: "",
  description: "",
  source: "",
  status: "Draft",
};

/** The form's values for a saved library, read from GraphQL or returned by a write. */
export function libraryFormValues(
  library: Pick<FormulaLibrary, "name" | "description" | "source" | "status">,
): FormulaLibraryFormValues {
  return {
    name: library.name,
    description: library.description ?? "",
    source: library.source,
    status: library.status,
  };
}

/**
 * Whether saving these values makes a new version of an approved or in-review library, which
 * returns it to draft: a change to its source or its name.
 */
export function saveDemotesLibrary(
  library: Pick<FormulaLibraryRecord, "status" | "source" | "name"> | null,
  values: Pick<FormulaLibraryFormValues, "source" | "name">,
): boolean {
  if (!library || (library.status !== "Active" && library.status !== "InReview")) return false;
  return library.source !== values.source || library.name !== values.name.trim();
}
