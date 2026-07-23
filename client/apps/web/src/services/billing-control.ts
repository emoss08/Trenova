import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  billingControlSchema,
  type BillingControl,
} from "@/types/billing-control";

export class BillingControlService {
  public async get() {
    const response = await api.get<BillingControl>("/billing-controls/");

    return safeParse(billingControlSchema, response, "Billing Control");
  }

  public async update(data: BillingControl) {
    const response = await api.put<BillingControl>("/billing-controls/", data);

    return safeParse(billingControlSchema, response, "Billing Control");
  }
}
