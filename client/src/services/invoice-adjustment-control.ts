import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  invoiceAdjustmentControlSchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";

export class InvoiceAdjustmentControlService {
  public async get() {
    const response = await api.get<InvoiceAdjustmentControl>("/invoice-adjustment-controls/");

    return safeParse(
      invoiceAdjustmentControlSchema,
      response,
      "Invoice Adjustment Control",
    );
  }

  public async update(data: InvoiceAdjustmentControl) {
    const response = await api.put<InvoiceAdjustmentControl>("/invoice-adjustment-controls/", data);

    return safeParse(
      invoiceAdjustmentControlSchema,
      response,
      "Invoice Adjustment Control",
    );
  }
}
