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

// Web socket constants
export const WEB_SOCKET_URL = import.meta.env.VITE_WS_URL;
// export const MAX_WEBSOCKET_RETRIES = import.meta.env.VITE_MAX_WEBSOCKET_RETRIES;
export const WEBSOCKET_RETRY_INTERVAL = import.meta.env
  .VITE_WEBSOCKET_RETRY_INTERVAL;
export const ENABLE_WEBSOCKETS = import.meta.env
  .VITE_ENABLE_WEBSOCKETS as boolean;

// API constants
export const API_URL = import.meta.env.VITE_API_URL as string;

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
