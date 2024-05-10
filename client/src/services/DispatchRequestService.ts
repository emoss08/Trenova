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
