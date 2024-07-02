import axios from "@/lib/axiosConfig";
import { Commodity, HazardousMaterial } from "@/types/commodities";

/**
 * Fetches hazardous material from the server.
 * @returns A promise that resolves to an array of hazardous material.
 */
export async function getHazardousMaterials(): Promise<HazardousMaterial[]> {
  const response = await axios.get("/hazardous-materials/");
  return response.data.results;
}

export async function getCommodities(): Promise<ReadonlyArray<Commodity>> {
  const response = await axios.get("commodities/");
  return response.data.results;
}
