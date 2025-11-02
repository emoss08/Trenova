import { http } from "@/lib/http-client";
import { FiscalYearSchema } from "@/lib/schemas/fiscal-year-schema";

export class FiscalYearAPI {
  async close(id: FiscalYearSchema["id"]) {
    const response = await http.put<FiscalYearSchema>(
      `/fiscal-years/${id}/close/`,
    );
    return response.data;
  }

  async lock(id: FiscalYearSchema["id"]) {
    const response = await http.put<FiscalYearSchema>(
      `/fiscal-years/${id}/lock/`,
    );
    return response.data;
  }

  async unlock(id: FiscalYearSchema["id"]) {
    const response = await http.put<FiscalYearSchema>(
      `/fiscal-years/${id}/unlock/`,
    );
    return response.data;
  }

  async activate(id: FiscalYearSchema["id"]) {
    const response = await http.put<FiscalYearSchema>(
      `/fiscal-years/${id}/activate/`,
    );
    return response.data;
  }
}
