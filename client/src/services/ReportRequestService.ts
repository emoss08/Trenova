import axios from "@/lib/axiosConfig";

/**
 * Fetches the columns for the specified table from the server.
 * @param tableName - The name of the table to fetch columns for.
 * @returns A promise that resolves to the columns of the table.
 */
export async function getColumns(tableName: string): Promise<any> {
  const response = await axios.get("/reports/column-names/", {
    params: {
      tableName: tableName,
    },
  });
  return response.data.results;
}
