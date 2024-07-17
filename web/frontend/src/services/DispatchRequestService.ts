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
