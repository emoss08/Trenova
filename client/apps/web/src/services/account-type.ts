import type { AccountTypeInput, AccountTypePatchInput } from "@trenova/graphql/generated/graphql";
import {
  BulkUpdateAccountTypeStatusDocument,
  CreateAccountTypeDocument,
  PatchAccountTypeDocument,
  UpdateAccountTypeDocument,
} from "@trenova/graphql/generated/documents/account-type/mutations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { AccountType, BulkUpdateAccountTypeStatusRequest } from "@/types/account-type";

export function toAccountTypeInput(data: AccountType): AccountTypeInput {
  return {
    status: data.status,
    code: data.code,
    name: data.name,
    description: data.description || null,
    category: data.category,
    color: data.color || null,
    version: data.version,
  };
}

export class AccountTypeService {
  public async create(data: AccountType) {
    const response = await requestGraphQL({
      document: CreateAccountTypeDocument,
      operationName: "CreateAccountType",
      variables: { input: toAccountTypeInput(data) },
    });

    return response.createAccountType;
  }

  public async update(id: string, data: AccountType) {
    const response = await requestGraphQL({
      document: UpdateAccountTypeDocument,
      operationName: "UpdateAccountType",
      variables: { id, input: toAccountTypeInput(data) },
    });

    return response.updateAccountType;
  }

  public async bulkUpdateStatus(request: BulkUpdateAccountTypeStatusRequest) {
    const response = await requestGraphQL({
      document: BulkUpdateAccountTypeStatusDocument,
      operationName: "BulkUpdateAccountTypeStatus",
      variables: { input: request },
    });

    return response.bulkUpdateAccountTypeStatus;
  }

  public async patch(id: string, input: AccountTypePatchInput) {
    const response = await requestGraphQL({
      document: PatchAccountTypeDocument,
      operationName: "PatchAccountType",
      variables: { id, input },
    });

    return response.patchAccountType;
  }
}

export const accountTypeService = new AccountTypeService();
