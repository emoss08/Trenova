import axios from "@/lib/axiosConfig";
import type {
  EquipmentManufacturer,
  EquipmentStatus,
  EquipmentType,
  Tractor,
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
