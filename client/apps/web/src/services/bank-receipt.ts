import {
  BankReceiptDocument,
  BankReceiptExceptionsDocument,
  BankReceiptMatchSuggestionsDocument,
  BankReceiptSummaryDocument,
  MatchBankReceiptDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { BankReceipt, MatchSuggestion, ReconciliationSummary } from "@/types/bank-receipt";

type RequestOptions = { signal?: AbortSignal };

export class BankReceiptService {
  async getSummary({ signal }: RequestOptions = {}): Promise<ReconciliationSummary> {
    const response = await requestGraphQL({
      document: BankReceiptSummaryDocument,
      operationName: "BankReceiptSummary",
      variables: {},
      signal,
    });
    return response.bankReceiptSummary;
  }

  /** The receipts no payment matched, newest first. */
  async getExceptions({ signal }: RequestOptions = {}): Promise<BankReceipt[]> {
    const response = await requestGraphQL({
      document: BankReceiptExceptionsDocument,
      operationName: "BankReceiptExceptions",
      variables: {},
      signal,
    });
    return response.bankReceiptExceptions;
  }

  async getById(id: string, { signal }: RequestOptions = {}): Promise<BankReceipt> {
    const response = await requestGraphQL({
      document: BankReceiptDocument,
      operationName: "BankReceipt",
      variables: { id },
      signal,
    });
    return response.bankReceipt;
  }

  async getSuggestions(id: string, { signal }: RequestOptions = {}): Promise<MatchSuggestion[]> {
    const response = await requestGraphQL({
      document: BankReceiptMatchSuggestionsDocument,
      operationName: "BankReceiptMatchSuggestions",
      variables: { id },
      signal,
    });
    return response.bankReceiptMatchSuggestions;
  }

  async match(id: string, customerPaymentId: string): Promise<BankReceipt> {
    const response = await requestGraphQL({
      document: MatchBankReceiptDocument,
      operationName: "MatchBankReceipt",
      variables: { id, paymentId: customerPaymentId },
    });
    return response.matchBankReceipt;
  }
}
