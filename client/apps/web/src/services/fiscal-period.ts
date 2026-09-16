import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  fiscalPeriodCloseCheckSchema,
  type FiscalPeriod,
  type FiscalPeriodCloseCheck,
} from "@/types/fiscal-period";

export class FiscalPeriodService {
  async activate(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/activate/`);
  }

  async close(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/close/`);
  }

  async closeBlockers(id: FiscalPeriod["id"]): Promise<FiscalPeriodCloseCheck> {
    const response = await api.get(`/fiscal-periods/${id}/close-blockers/`);
    return safeParse(fiscalPeriodCloseCheckSchema, response, "Fiscal Period Close Check");
  }

  async reopen(id: FiscalPeriod["id"], reopenReason: string) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/reopen/`, { reopenReason });
  }

  async lock(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/lock/`);
  }

  async unlock(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/unlock/`);
  }
}
