import axios from "@/lib/axiosConfig";
import {
  AccessorialCharge,
  BillingQueue,
  ChargeType,
  DocumentClassification,
  OrdersReadyProps,
} from "@/types/billing";

/**
 * Fetches the details of a charge type with the specified ID.
 * @param id - The ID of the charge type to fetch details for.
 * @returns A promise that resolves to the charge type's details.
 */
export async function getChargeTypeDetails(id: string): Promise<ChargeType> {
  const response = await axios.get(`/charge_types/${id}/`);
  return response.data;
}

/**
 * Fetches accessorial charges from the server.
 * @returns A promise that resolves to an array of accessorial charges.
 */
export async function getAccessorialCharges(): Promise<AccessorialCharge[]> {
  const response = await axios.get("/accessorial_charges/");
  return response.data.results;
}

/**
 * Fetches the details of the accessorial charge with the specified ID.
 * @param id - The ID of the accessorial charge to fetch details for.
 * @returns A promise that resolves to the accessorial charge's details.
 */
export async function getAccessorialChargeDetails(
  id: string,
): Promise<AccessorialCharge> {
  const response = await axios.get(`/accessorial_charges/${id}/`);
  return response.data;
}

/**
 * Fetches orders ready to be billed from the server.
 * @returns A promise that resolves to an array of orders ready to be billed.
 */
export async function getOrdersReadyToBill(): Promise<OrdersReadyProps[]> {
  const response = await axios.get("/billing/orders_ready");
  return response.data.results;
}

/**
 * Fetches billing queue from the server.
 * @returns A promise that resolves to an array of billing queue records.
 */
export async function getBillingQueue(): Promise<BillingQueue[]> {
  const response = await axios.get("/billing_queue/");
  return response.data.results;
}

export async function getDocumentClassifications(): Promise<
  DocumentClassification[]
> {
  const response = await axios.get("/document_classifications/");
  return response.data.results;
}
