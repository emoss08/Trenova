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
