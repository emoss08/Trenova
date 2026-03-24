import { api } from "@/lib/api";
import type { FiscalPeriod } from "@/types/fiscal-period";

export class FiscalPeriodService {
  async close(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/close/`);
  }

  async reopen(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/reopen/`);
  }

  async lock(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/lock/`);
  }

  async unlock(id: FiscalPeriod["id"]) {
    return api.put<FiscalPeriod>(`/fiscal-periods/${id}/unlock/`);
  }
}
