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
