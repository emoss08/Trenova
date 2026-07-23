import { api } from "@/lib/api";
import type { FiscalYear } from "@/types/fiscal-year";

export class FiscalYearService {
  async close(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/close/`);
  }

  async lock(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/lock/`);
  }

  async unlock(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/unlock/`);
  }

  async activate(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/activate/`);
  }
}
