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
import { AssignTractorPayload } from "@/types/equipment";
import { ApiResponse } from "@/types/server";
import type {
  FormulaTemplate,
  HazardousMaterialSegregationRule,
  ServiceType,
  Shipment,
  ShipmentStatus,
  ShipmentType,
} from "@/types/shipment";

/**
 * Fetches the shipments from the server.
 * @returns A promise that resolves to a FeasibilityToolControl object.
 */
export async function getShipments(
  searchQuery: string,
  statusFilter: string,
  offset: number,
  limit: number,
): Promise<ApiResponse<Shipment>> {
  const response = await axios.get("/shipments/", {
    params: {
      search: searchQuery,
      status: statusFilter,
      limit: limit,
      offset: offset,
    },
  });
  return response.data;
}

/** Type for the response of the getShipmentCountByStatus function. */
type ShipmentCount = {
  status: ShipmentStatus;
  count: number;
};

/**
 * Fetches the shipment count by status from the server.
 * @param searchQuery The search query to filter the shipments.
 * @returns A promise that resolves to a ShipmentCount object.
 */
export async function getShipmentCountByStatus(
  searchQuery?: string,
  statusFilter?: string,
): Promise<ApiResponse<ShipmentCount>> {
  const response = await axios.get("/shipments/count/", {
    params: {
      search: searchQuery,
      status: statusFilter,
    },
  });
  return response.data;
}

/**
 * Fetches the next pro number from the server.
 * @param proNumber The pro number of the shipment.
 * @returns A promise that resolves to a Shipment object.
 */
export async function getNextProNumber(): Promise<string> {
  const response = await axios.get("/shipments/get_new_pro_number/");
  return response.data.proNumber;
}

/**
 * Fetches the formula templates from the server.
 * @returns A promise that resolves to a FormulaTemplate object.
 */
export async function getFormulaTemplates(): Promise<FormulaTemplate[]> {
  const response = await axios.get("/formula-templates/");
  return response.data.results;
}

/**
 * Fetches the service types from the server.
 * @returns A promise that resolves to a ServiceType object.
 */
export async function getServiceTypes(): Promise<ServiceType[]> {
  const response = await axios.get("/service-types/");
  return response.data.results;
}

/**
 * Fetches and validates the BOL number from the server.
 * @param bol_number The BOL number of the shipment.
 * @returns A promise that resolves to a boolean and a message value.
 * @throws An error if the BOL number is invalid.
 */
export async function validateBOLNumber(
  bol_number: string,
): Promise<{ valid: boolean; message: string }> {
  const response = await axios.post("/shipments/check_duplicate_bol/", {
    bol_number,
  });

  return response.data;
}

/**
 * Fetches shipment types from the server.
 * @returns A promise that resolves to an array of shipment types.
 */
export async function getShipmentTypes(): Promise<ReadonlyArray<ShipmentType>> {
  const response = await axios.get("/shipment-types/");
  return response.data.results;
}

/**
 * Fetches Hazardous material segregation rules from the server.
 * @returns A Promise that resolves to an array of HazardousMaterialSegregationRule objects.
 */

export async function getHazardousSegregationRules(): Promise<
  ReadonlyArray<HazardousMaterialSegregationRule>
> {
  const response = await axios.get("/hazardous-material-segregations/");
  return response.data.results;
}

export async function assignTractorToShipment(
  payload: AssignTractorPayload,
): Promise<{ message: string }> {
  const response = await axios.post("/shipments/assign-tractor/", payload);
  return response.data;
}
