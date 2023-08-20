/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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

import axios from "@/lib/AxiosConfig";
import {
  AccessorialCharge,
  BillingQueue,
  ChargeType,
  DocumentClassification,
  OrdersReadyProps,
} from "@/types/apps/billing";

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
