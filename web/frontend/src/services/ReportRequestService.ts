import axios from "@/lib/axiosConfig";

export type ReportColumn = {
  label: string;
  value: string;
  description: string;
};

type Relationship = {
  foreignKey: string;
  referencedTable: string;
  referencedColumn: string;
  columns: ReportColumn[];
};

type GetColumnNameReponse = {
  columns: ReportColumn[];
  relationships: Relationship[];
};

/**
 * Fetches the columns for the specified table from the server.
 * @param tableName - The name of the table to fetch columns for.
 * @returns A promise that resolves to the columns of the table.
 */
export async function getColumns(
  tableName: string,
): Promise<GetColumnNameReponse> {
  const response = await axios.get("/reports/column-names/", {
    params: {
      tableName: tableName,
    },
  });
  return response.data.results;
}
