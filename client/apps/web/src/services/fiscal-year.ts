import { api } from "@trenova/shared/lib/api";
import type { FiscalYear } from "@/types/fiscal-year";

export class FiscalYearService {
  async close(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/close/`);
  }

  async activate(id: FiscalYear["id"]) {
    return api.put<FiscalYear>(`/fiscal-years/${id}/activate/`);
  }
}
