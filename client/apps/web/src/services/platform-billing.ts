import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { billingSummarySchema, type BillingSummary } from "@/types/platform-billing";

export class PlatformBillingService {
  public async getSummary() {
    const response = await api.get<BillingSummary>("/me/billing");

    return safeParse(billingSummarySchema, response, "Subscription and usage");
  }
}
