import {
  AssignBankReceiptWorkItemDocument,
  BankReceiptWorkItemDocument,
  BankReceiptWorkItemsDocument,
  DismissBankReceiptWorkItemDocument,
  ResolveBankReceiptWorkItemDocument,
  StartBankReceiptWorkItemReviewDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BankReceiptWorkItem, ResolutionType } from "@/types/bank-receipt-work-item";

type RequestOptions = { signal?: AbortSignal };

export class BankReceiptWorkItemService {
  /** The work items not yet resolved or dismissed, newest first. */
  async list({ signal }: RequestOptions = {}): Promise<BankReceiptWorkItem[]> {
    const response = await requestGraphQL({
      document: BankReceiptWorkItemsDocument,
      operationName: "BankReceiptWorkItems",
      variables: {},
      signal,
    });
    return response.bankReceiptWorkItems;
  }

  async getById(id: string, { signal }: RequestOptions = {}): Promise<BankReceiptWorkItem> {
    const response = await requestGraphQL({
      document: BankReceiptWorkItemDocument,
      operationName: "BankReceiptWorkItem",
      variables: { id },
      signal,
    });
    return response.bankReceiptWorkItem;
  }

  async assign(id: string, userId: string): Promise<BankReceiptWorkItem> {
    const response = await requestGraphQL({
      document: AssignBankReceiptWorkItemDocument,
      operationName: "AssignBankReceiptWorkItem",
      variables: { id, assignedToUserId: userId },
    });
    return response.assignBankReceiptWorkItem;
  }

  async startReview(id: string): Promise<BankReceiptWorkItem> {
    const response = await requestGraphQL({
      document: StartBankReceiptWorkItemReviewDocument,
      operationName: "StartBankReceiptWorkItemReview",
      variables: { id },
    });
    return response.startBankReceiptWorkItemReview;
  }

  async resolve(
    id: string,
    data: { resolutionType: ResolutionType; resolutionNote: string },
  ): Promise<BankReceiptWorkItem> {
    const response = await requestGraphQL({
      document: ResolveBankReceiptWorkItemDocument,
      operationName: "ResolveBankReceiptWorkItem",
      variables: { id, input: data },
    });
    return response.resolveBankReceiptWorkItem;
  }

  async dismiss(id: string, data: { resolutionNote: string }): Promise<BankReceiptWorkItem> {
    const response = await requestGraphQL({
      document: DismissBankReceiptWorkItemDocument,
      operationName: "DismissBankReceiptWorkItem",
      variables: { id, resolutionNote: data.resolutionNote },
    });
    return response.dismissBankReceiptWorkItem;
  }
}
