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
import { Worker } from "@/types/worker";

/**
 * Fetches an array of all workers from the server.
 * @param {number} limit The maximum number of workers to return.
 * @param searchQuery
 * @param fleetFilter
 * @returns {Promise<Worker[]>} A promise that resolves to an array of workers.
 */
export async function getWorkers(
  searchQuery?: string,
  fleetFilter?: string,
  limit: number = 100,
  status?: string,
): Promise<Worker[]> {
  const response = await axios.get("/workers/", {
    params: {
      limit,
      search: searchQuery,
      status: status,
      fleet_code: fleetFilter,
    },
  });
  return response.data.results;
}
