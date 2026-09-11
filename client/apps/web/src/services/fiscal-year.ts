import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  fiscalYearClosePlanSchema,
  type FiscalYear,
  type FiscalYearClosePlan,
} from "@/types/fiscal-year";

export class FiscalYearService {
  async close(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/close/`);
  }

  async closePreview(id: FiscalYear["id"]): Promise<FiscalYearClosePlan> {
    const response = await api.get(`/fiscal-years/${id}/close-preview/`);
    return safeParse(fiscalYearClosePlanSchema, response, "Fiscal Year Close Plan");
  }

  async reopen(id: FiscalYear["id"], reopenReason: string) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/reopen/`, { reopenReason });
  }

  async activate(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/activate/`);
  }
}
