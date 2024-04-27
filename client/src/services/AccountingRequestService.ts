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
  const response = await axios.get("/division_codes/", {
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
  const response = await axios.get(`/division_codes/${id}/`);
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
  const response = await axios.get("/revenue_codes/");
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
