/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
