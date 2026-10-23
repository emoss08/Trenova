import {
  FormulaTemplateTablesDocument,
  type FormulaTemplateTablesQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type FormulaTemplateTable = NonNullable<
  FormulaTemplateTablesQuery["formulaTemplate"]
>["tables"][number];

/** The rate tables a formula template reads, each with its pinned and newest published version. */
export async function fetchFormulaTemplateTables(
  templateId: string,
  options?: { signal?: AbortSignal },
): Promise<FormulaTemplateTable[]> {
  const data = await requestGraphQL({
    document: FormulaTemplateTablesDocument,
    operationName: "FormulaTemplateTables",
    variables: { id: templateId },
    signal: options?.signal,
  });
  return data.formulaTemplate?.tables ?? [];
}

/** The tables a newer published version is out for than the one the template pins. */
export function staleTablePins(tables: readonly FormulaTemplateTable[]): FormulaTemplateTable[] {
  return tables.filter(
    (table) => !table.archived && table.latestVersionNumber > table.versionNumber,
  );
}
