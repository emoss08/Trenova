import axios from "@/lib/axiosConfig";
import type {
  AccessorialCharge,
  BillingQueue,
  DocumentClassification,
  OrdersReadyProps,
} from "@/types/billing";

/**
 * Fetches accessorial charges from the server.
 * @returns A promise that resolves to an array of accessorial charges.
 */
export async function getAccessorialCharges(): Promise<AccessorialCharge[]> {
  const response = await axios.get("/accessorial-charges/");
  return response.data.results;
}

/**
 * Fetches orders ready to be billed from the server.
 * @returns A promise that resolves to an array of orders ready to be billed.
 */
export async function getOrdersReadyToBill(): Promise<OrdersReadyProps[]> {
  const response = await axios.get("/billing/orders-ready");
  return response.data.results;
}

/**
 * Fetches billing queue from the server.
 * @returns A promise that resolves to an array of billing queue records.
 */
export async function getBillingQueue(): Promise<BillingQueue[]> {
  const response = await axios.get("/billing-queue/");
  return response.data.results;
}

export async function getDocumentClassifications(): Promise<
  DocumentClassification[]
> {
  const response = await axios.get("/document-classifications/");
  return response.data.results;
}
