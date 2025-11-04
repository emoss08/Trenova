import { http } from "@/lib/http-client";
import { FiscalPeriodSchema } from "@/lib/schemas/fiscal-period-schema";

export class FiscalPeriodAPI {
  async close(id: FiscalPeriodSchema["id"]) {
    const response = await http.put<FiscalPeriodSchema>(
      `/fiscal-periods/${id}/close/`,
    );
    return response.data;
  }

  async reopen(id: FiscalPeriodSchema["id"]) {
    const response = await http.put<FiscalPeriodSchema>(
      `/fiscal-periods/${id}/reopen/`,
    );
    return response.data;
  }

  async lock(id: FiscalPeriodSchema["id"]) {
    const response = await http.put<FiscalPeriodSchema>(
      `/fiscal-periods/${id}/lock/`,
    );
    return response.data;
  }

  async unlock(id: FiscalPeriodSchema["id"]) {
    const response = await http.put<FiscalPeriodSchema>(
      `/fiscal-periods/${id}/unlock/`,
    );
    return response.data;
  }
}
