import { http } from "@/lib/http-client";

export async function bulkTransfer() {
  const response = await http.post(`/billing-queue/bulk-transfer`);
  return response.data as Promise<void>;
}
