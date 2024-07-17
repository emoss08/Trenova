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
import type {
  EquipmentManufacturer,
  EquipmentStatus,
  EquipmentType,
  Tractor,
  TractorAssignment,
  Trailer,
} from "@/types/equipment";
import { type ApiResponse } from "@/types/server";

/**
 * Get equipment types from the server.
 * @returns a list of equipment types
 */
export async function getEquipmentTypes(
  limit?: number,
): Promise<ReadonlyArray<EquipmentType>> {
  const response = await axios.get("equipment-types/", {
    params: {
      limit: limit,
    },
  });
  return response.data.results;
}

/**
 * Get equipment manufacturers from the server.
 * @returns a list of equipment manufacturers
 */
export async function getEquipmentManufacturers(
  limit?: number,
): Promise<ReadonlyArray<EquipmentManufacturer>> {
  const response = await axios.get("equipment-manufacturers/", {
    params: {
      limit: limit,
    },
  });
  return response.data.results;
}

/**
 * Get trailers from the server
 * @returns a list of trailers
 */
export async function getTrailers(limit?: number): Promise<Trailer[]> {
  const response = await axios.get("trailers", {
    params: {
      limit: limit,
      status: "A",
    },
  });
  return response.data.results;
}

/**
 * Get Tractors from the server
 * @returns a list of tractors
 */
export async function getTractors(
  status?: EquipmentStatus,
  offset?: number,
  limit?: number,
  fleetCodeId?: string,
  expandEquipDetails?: boolean,
  expandWorkerDetails?: boolean,
): Promise<ApiResponse<Tractor>> {
  const response = await axios.get("tractors", {
    params: {
      status: status,
      offset: offset,
      limit: limit,
      fleetCodeId: fleetCodeId,
      expandEquipDetails: expandEquipDetails,
      expandWorkerDetails: expandWorkerDetails,
    },
  });

  return response.data;
}

export async function getActiveAssignmentsForTractor(
  tractorId: string,
): Promise<TractorAssignment[]> {
  const response = await axios.get(`tractors/${tractorId}/assignments`);
  return response.data;
}
