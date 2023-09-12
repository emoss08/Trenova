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

// User info constants
import { BChoiceProps, IChoiceProps } from "@/types";

export const USER_ID_KEY = import.meta.env.VITE_USER_ID_KEY;
export const ORGANIZATION_ID_KEY = import.meta.env.VITE_ORGANIZATION_ID_KEY;

// Web socket constants
export const WEB_SOCKET_URL = import.meta.env.VITE_WS_URL;
// export const MAX_WEBSOCKET_RETRIES = import.meta.env.VITE_MAX_WEBSOCKET_RETRIES;
export const WEBSOCKET_RETRY_INTERVAL = import.meta.env
  .VITE_WEBSOCKET_RETRY_INTERVAL;
export const ENABLE_WEBSOCKETS = import.meta.env.VITE_ENABLE_WEBSOCKETS;

// API constants
export const API_URL = import.meta.env.VITE_API_URL as string;

/**
 * Retrieves the current user's ID from session storage.
 * @returns The user's ID, or null if it was not found.
 */
export const getUserId = (): string | null => {
  const userId = sessionStorage.getItem(USER_ID_KEY);
  if (userId) {
    return userId;
  }
  return null;
};

/**
 * Transforms the first character of the provided string to upper case.
 * @param str - The string to be transformed.
 * @returns A new string with the first character in upper case and the rest of the string unchanged.
 */
export function upperFirst(str: string) {
  return str.charAt(0).toUpperCase() + str.slice(1);
}

/**
 * Returns status choices for a select input.
 */
type TStatusChoiceProps = "A" | "I";

/**
 * Returns status choices for a select input.
 * @returns An array of status choices.
 */
export const statusChoices: IChoiceProps<TStatusChoiceProps>[] = [
  { value: "A", label: "Active" },
  { value: "I", label: "Inactive" },
];

/**
 * Returns boolean yes & no choices for a select input.
 * @returns An array of yes & no choices.
 */
export const yesAndNoChoicesBoolean: ReadonlyArray<BChoiceProps> = [
  { value: true, label: "Yes" },
  { value: false, label: "No" },
];

/**
 * Returns yes & no choices for a select input.
 */
type TYesNoChoiceProps = "Y" | "N";

/**
 * Returns yes & no choices for a select input.
 * @returns An array of yes & no choices.
 */
export const yesAndNoChoices: IChoiceProps<TYesNoChoiceProps>[] = [
  { value: "Y", label: "Yes" },
  { value: "N", label: "No" },
];

/**
 * Returns rate method choices for a select input
 */
export type TRateMethodChoices = "F" | "PM" | "PS" | "PP" | "O";

export const rateMethodChoices: ReadonlyArray<
  IChoiceProps<TRateMethodChoices>
> = [
  { value: "F", label: "Flat" },
  { value: "PM", label: "Per Mile" },
  { value: "PS", label: "Per Stop" },
  { value: "PP", label: "Per Pound" },
  { value: "O", label: "Other" },
];

/**
 * Truncates the provided text to the specified limit.
 * @param text - The text to be truncated.
 * @param limit - The maximum number of characters to be displayed.
 * @returns The truncated text.
 */
export function truncateText(text: string, limit: number): string {
  return text.length > limit ? `${text.substring(0, limit)}...` : text;
}

/**
 * Formats the provided amount as a US dollar amount.
 * @param amount - The amount to be formatted.
 * @constructor - The formatted amount.
 */
export function USDollarFormat(amount: number): string {
  return amount.toLocaleString("en-US", {
    style: "currency",
    currency: "USD",
  });
}
