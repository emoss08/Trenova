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
