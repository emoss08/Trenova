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
import {
  AccountingControl,
  DivisionCode,
  GeneralLedgerAccount,
  RevenueCode,
  Tag,
} from "@/types/accounting";

/**
 * Fetches active division codes from the server.
 * @returns A promise that resolves to an array of division codes.
 */
export async function getDivisionCodes(): Promise<DivisionCode[]> {
  const response = await axios.get("/division-codes/", {
    params: {
      status: "A",
    },
  });
  return response.data.results;
}

/**
 * Fetches the details of the division code with the specified ID.
 * @param id - The ID of the division code to fetch details for.
 * @returns A promise that resolves to the division code's details.
 */
export async function getDivisionCodeDetail(id: string): Promise<DivisionCode> {
  const response = await axios.get(`/division-codes/${id}/`);
  return response.data;
}

/**
 * Fetches active general ledger accounts from the server.
 * @returns A promise that resolves to an array of general ledger accounts.
 */
export async function getGLAccounts(): Promise<GeneralLedgerAccount[]> {
  const response = await axios.get("/general-ledger-accounts/", {
    params: {
      status: "A",
    },
  });
  return response.data.results;
}

export async function getRevenueCodes(): Promise<RevenueCode[]> {
  const response = await axios.get("/revenue-codes/");
  return response.data.results;
}

/**
 * Fetches tags accounts from the server.
 * @returns A promise that resolves to an array of tags.
 */
export async function getTags(): Promise<Tag[]> {
  const response = await axios.get("/tags/");
  return response.data.results;
}

/**
 * Fetches accounting control from the server.
 * @returns A promise that resolves to an array of accounting control.
 * @note This should only return one result.
 */
export async function getAccountingControl(): Promise<AccountingControl> {
  const response = await axios.get("/accounting-control/");
  return response.data;
}
