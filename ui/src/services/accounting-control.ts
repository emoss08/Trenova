import { http } from "@/lib/http-client";
import type { AccountingControlSchema } from "@/lib/schemas/accounting-control-schema";

export class AccountingControlAPI {
  async get() {
    const response = await http.get<AccountingControlSchema>(
      "/accounting-controls/",
    );
    return response.data;
  }

  async update(data: AccountingControlSchema) {
    const response = await http.put<AccountingControlSchema>(
      `/accounting-controls/`,
      data,
    );
    return response.data;
  }
}
