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
  CommentType,
  FeasibilityToolControl,
  FleetCode,
  Rate,
} from "@/types/dispatch";

/**
 * Fetches new Rate Number from the server.
 * @returns A promise that resolves to a string representation of the latest rate number.
 */
export async function getNewRateNumber(): Promise<string> {
  const response = await axios.get("/rates/get-new-rate-number/");
  return response.data.rateNumber;
}

/**
 * Fetches the feasibility tool control from the server.
 * @returns A promise that resolves to a FeasibilityToolControl object.
 */
export async function getFeasibilityControl(): Promise<FeasibilityToolControl> {
  const response = await axios.get("/feasibility-tool-control/");
  return response.data;
}

/**
 * Fetches the comment types from the server.
 * @returns A promise that resolves to a CommentType object.
 */
export async function getCommentTypes(): Promise<CommentType[]> {
  const response = await axios.get("/comment-types/");
  return response.data.results;
}

/**
 * Fetches the fleet codes from the server.
 * @param limit The maximum number of fleet codes to return.
 * @returns A promise that resolves to a FleetCode object.
 */
export async function getFleetCodes(limit?: number): Promise<FleetCode[]> {
  const response = await axios.get("/fleet-codes/", {
    params: {
      status: "A",
      limit: limit,
    },
  });
  return response.data.results;
}

/**
 * Fetches the rates from the server.
 * @param limit The maximum number of rates to return.
 * @returns A promise that resolves to a Rate object.
 */
export async function getRates(limit?: number): Promise<Rate[]> {
  const response = await axios.get("/rates/", {
    params: {
      status: "A",
      limit: limit,
    },
  });
  return response.data.results;
}
