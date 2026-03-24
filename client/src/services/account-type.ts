import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  bulkUpdateAccountTypeStatusResponseSchema,
  accountTypeSchema,
  type BulkUpdateAccountTypeStatusRequest,
  type BulkUpdateAccountTypeStatusResponse,
  type AccountType,
} from "@/types/account-type";

export class AccountTypeService {
  public async bulkUpdateStatus(request: BulkUpdateAccountTypeStatusRequest) {
    const response = await api.post<BulkUpdateAccountTypeStatusResponse>(
      "/account-types/bulk-update-status/",
      request,
    );

    return safeParse(bulkUpdateAccountTypeStatusResponseSchema, response, "Bulk Update Account Type Status");
  }

  public async patch(id: AccountType["id"], data: Partial<AccountType>) {
    const response = await api.patch<AccountType>(
      `/account-types/${id}/`,
      data,
    );

    return safeParse(accountTypeSchema, response, "Account Type");
  }
}
