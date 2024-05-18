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
