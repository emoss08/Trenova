import axios from "@/lib/axiosConfig";
import { Customer } from "@/types/customer";

export async function getCustomers(): Promise<ReadonlyArray<Customer>> {
  const response = await axios.get("customers/");
  return response.data.results;
}
