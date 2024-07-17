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
