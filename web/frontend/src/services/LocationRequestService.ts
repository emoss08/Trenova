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
  GoogleAutoCompleteResult,
  Location,
  LocationCategory,
  MonthlyPickupData,
  USStates,
} from "@/types/location";

/**
 * Fetches locations from the server.
 * @returns A promise that resolves to an array of locations.
 */
export async function getLocations(
  locationStatus: string,
): Promise<Location[]> {
  const response = await axios.get("/locations/", {
    params: {
      status: locationStatus,
    },
  });
  return response.data.results;
}

/**
 * Fetches location pickup data from the server.
 * @param locationId The location id to fetch pickup data for.
 * @returns A promise that resolves to an array of monthly pickup data.
 */
export async function getLocationPickupData(
  locationId: string,
): Promise<MonthlyPickupData[]> {
  const response = await axios.get(
    `/locations/${locationId}/monthly_pickup_data`,
  );
  return response.data;
}

/**
 * Fetches location categories from the server.
 * @returns A promise that resolves to an array of location categories.
 */
export async function getLocationCategories(): Promise<LocationCategory[]> {
  const response = await axios.get("/location-categories/");
  return response.data.results;
}

/**
 * Fetches US states from the server.
 * @param limit The number of states to fetch.
 * @returns A promise that resolves to an array of US states.
 */
export async function getUSStates(limit: number = 100): Promise<USStates[]> {
  const response = await axios.get("/us-states/", {
    params: {
      limit: limit,
    },
  });
  return response.data.results;
}

/**
 * Fetches auto completed location results from the server.
 * @param searchQuery The search query to use for the auto complete.
 * @returns A promise that resolves to an array of auto complete results.
 */
export async function searchLocation(
  searchQuery: string,
): Promise<GoogleAutoCompleteResult> {
  const response = await axios.get("/locations/autocomplete/", {
    params: {
      query: searchQuery,
    },
  });

  return response.data;
}
