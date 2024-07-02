import axios from "@/lib/axiosConfig";
import { Worker } from "@/types/worker";

/**
 * Fetches an array of all workers from the server.
 * @param {number} limit The maximum number of workers to return.
 * @param searchQuery
 * @param fleetFilter
 * @returns {Promise<Worker[]>} A promise that resolves to an array of workers.
 */
export async function getWorkers(
  searchQuery?: string,
  fleetFilter?: string,
  limit: number = 100,
  status?: string,
): Promise<Worker[]> {
  const response = await axios.get("/workers/", {
    params: {
      limit,
      search: searchQuery,
      status: status,
      fleet_code: fleetFilter,
    },
  });
  return response.data.results;
}
