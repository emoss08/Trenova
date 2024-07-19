/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

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
