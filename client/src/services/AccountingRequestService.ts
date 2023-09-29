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
  DivisionCode,
  GeneralLedgerAccount,
  RevenueCode,
} from "@/types/accounting";

/**
 * Fetches division codes from the server.
 * @returns A promise that resolves to an array of division codes.
 */
export async function getDivisionCodes(): Promise<DivisionCode[]> {
  const response = await axios.get("/division_codes/");
  return response.data.results;
}

/**
 * Fetches the details of the division code with the specified ID.
 * @param id - The ID of the division code to fetch details for.
 * @returns A promise that resolves to the division code's details.
 */
export async function getDivisionCodeDetail(id: string): Promise<DivisionCode> {
  const response = await axios.get(`/division_codes/${id}/`);
  return response.data;
}

/**
 * Fetches general ledger accounts from the server.
 * @returns A promise that resolves to an array of general ledger accounts.
 */
export async function getGLAccounts(): Promise<GeneralLedgerAccount[]> {
  const response = await axios.get("/gl_accounts/");
  return response.data.results;
}

/**
 * Fetches the details of the general ledger account with the specified ID.
 * @param id
 */
export async function getGLAccountDetail(
  id: string,
): Promise<GeneralLedgerAccount> {
  const response = await axios.get(`/gl_accounts/${id}/`);
  return response.data;
}

/**
 * Fetches the details of the revenue code with the specified ID.
 * @param id
 */
export async function getRevenueCodeDetail(id: string): Promise<RevenueCode> {
  const response = await axios.get(`/revenue_codes/${id}/`);
  return response.data;
}
