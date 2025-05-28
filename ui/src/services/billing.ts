import { http } from "@/lib/http-client";

export class BillingAPI {
  async bulkTransfer() {
    const response = await http.post(`/billing-queue/bulk-transfer`);
    return response.data as Promise<void>;
  }
}
