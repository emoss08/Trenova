import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  accountingControlSchema,
  type AccountingControl,
} from "@/types/accounting-control";

export class AccountingControlService {
  public async get() {
    const response = await api.get<AccountingControl>(
      "/accounting-controls/",
    );

    return safeParse(accountingControlSchema, response, "Accounting Control");
  }

  public async update(data: AccountingControl) {
    const response = await api.put<AccountingControl>(
      "/accounting-controls/",
      data,
    );

    return safeParse(accountingControlSchema, response, "Accounting Control");
  }
}
