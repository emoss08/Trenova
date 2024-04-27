import axios from "@/lib/axiosConfig";
import type {
  FormulaTemplate,
  HazardousMaterialSegregationRule,
  ServiceType,
  Shipment,
  ShipmentType,
} from "@/types/shipment";

/**
 * Fetches the shipments from the server.
 * @returns A promise that resolves to a FeasibilityToolControl object.
 */
export async function getShipments(
  searchQuery?: string,
  statusFilter?: string,
): Promise<Shipment[]> {
  const response = await axios.get("/shipments/", {
    params: {
      search: searchQuery,
      status: statusFilter,
    },
  });
  return response.data.results;
}

/** Type for the response of the getShipmentCountByStatus function. */
type ShipmentCount = {
  status: string;
  count: number;
};

/** Type for the response of the getShipmentCountByStatus function. */
type ShipmentsByStatusResponse = {
  results: ShipmentCount[];
  totalCount: number;
};

/**
 * Fetches the shipment count by status from the server.
 * @param searchQuery The search query to filter the shipments.
 * @returns A promise that resolves to a ShipmentCount object.
 */
export async function getShipmentCountByStatus(
  searchQuery?: string,
): Promise<ShipmentsByStatusResponse> {
  const response = await axios.get("/shipments/get_shipment_count_by_status/", {
    params: {
      search: searchQuery,
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
  const response = await axios.get("/formula_templates/");
  return response.data.results;
}

/**
 * Fetches the service types from the server.
 * @returns A promise that resolves to a ServiceType object.
 */
export async function getServiceTypes(): Promise<ServiceType[]> {
  const response = await axios.get("/service_types/");
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
  const response = await axios.get("/shipment_types/");
  return response.data.results;
}

/**
 * Fetches Hazardous material segregation rules from the server.
 * @returns A Promise that resolves to an array of HazardousMaterialSegregationRule objects.
 */

export async function getHazardousSegregationRules(): Promise<
  ReadonlyArray<HazardousMaterialSegregationRule>
> {
  const response = await axios.get("/hazardous_material_segregation_rules/");
  return response.data.results;
}
