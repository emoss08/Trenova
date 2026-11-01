import {
  BankReceiptBatchDocument,
  BankReceiptBatchesDocument,
  ImportBankReceiptBatchDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  BankReceiptBatch,
  BatchDetailResponse,
  CreateBatchRequest,
} from "@/types/bank-receipt-batch";

type RequestOptions = { signal?: AbortSignal };

export class BankReceiptBatchService {
  async list({ signal }: RequestOptions = {}): Promise<BankReceiptBatch[]> {
    const response = await requestGraphQL({
      document: BankReceiptBatchesDocument,
      operationName: "BankReceiptBatches",
      variables: {},
      signal,
    });
    return response.bankReceiptBatches;
  }

  async getById(batchId: string, { signal }: RequestOptions = {}): Promise<BatchDetailResponse> {
    const response = await requestGraphQL({
      document: BankReceiptBatchDocument,
      operationName: "BankReceiptBatch",
      variables: { id: batchId },
      signal,
    });
    return response.bankReceiptBatch;
  }

  async create(data: CreateBatchRequest): Promise<BatchDetailResponse> {
    const response = await requestGraphQL({
      document: ImportBankReceiptBatchDocument,
      operationName: "ImportBankReceiptBatch",
      variables: {
        input: {
          source: data.source,
          reference: data.reference,
          receipts: data.receipts.map((receipt) => ({
            receiptDate: receipt.receiptDate,
            amountMinor: receipt.amountMinor,
            referenceNumber: receipt.referenceNumber,
            memo: receipt.memo ?? null,
          })),
        },
      },
    });
    return response.importBankReceiptBatch;
  }
}
