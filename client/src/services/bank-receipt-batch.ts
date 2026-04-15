import { api } from "@/lib/api";
import type {
  BankReceiptBatch,
  BatchDetailResponse,
  CreateBatchRequest,
} from "@/types/bank-receipt-batch";

export class BankReceiptBatchService {
  async list() {
    return api.get<BankReceiptBatch[]>("/accounting/bank-receipt-batches/");
  }

  async getById(batchId: string) {
    return api.get<BatchDetailResponse>(`/accounting/bank-receipt-batches/${batchId}/`);
  }

  async create(data: CreateBatchRequest) {
    return api.post<BatchDetailResponse>("/accounting/bank-receipt-batches/", data);
  }
}
