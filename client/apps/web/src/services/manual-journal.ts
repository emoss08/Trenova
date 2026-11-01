import {
  ApproveManualJournalDocument,
  CancelManualJournalDocument,
  CreateManualJournalDraftDocument,
  ManualJournalDocument,
  PostManualJournalDocument,
  RejectManualJournalDocument,
  SubmitManualJournalDocument,
  UpdateManualJournalDraftDocument,
  type ManualJournalDraftInput,
  type ManualJournalQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { ManualJournal, ManualJournalDraftValues } from "@/types/manual-journal";

type RequestOptions = { signal?: AbortSignal };

type ManualJournalResult = Omit<NonNullable<ManualJournalQuery["manualJournal"]>, "lines"> & {
  lines?: NonNullable<ManualJournalQuery["manualJournal"]>["lines"];
};

function toManualJournal(raw: ManualJournalResult): ManualJournal {
  return {
    ...raw,
    reason: raw.reason ?? "",
    rejectionReason: raw.rejectionReason ?? undefined,
    cancelReason: raw.cancelReason ?? undefined,
    createdById: raw.createdById ?? undefined,
    updatedById: raw.updatedById ?? undefined,
    lines: raw.lines ?? undefined,
  };
}

export function toManualJournalDraftInput(
  values: ManualJournalDraftValues,
): ManualJournalDraftInput {
  return {
    description: values.description,
    reason: values.reason || null,
    accountingDate: values.accountingDate,
    currencyCode: values.currencyCode,
    lines: values.lines.map((line) => ({
      glAccountId: line.glAccountId,
      description: line.description,
      debitAmount: line.debitAmount,
      creditAmount: line.creditAmount,
      customerId: line.customerId || null,
      locationId: line.locationId || null,
    })),
  };
}

export class ManualJournalService {
  async getById(id: string, { signal }: RequestOptions = {}) {
    const response = await requestGraphQL({
      document: ManualJournalDocument,
      operationName: "ManualJournal",
      variables: { id },
      signal,
    });
    return response.manualJournal ? toManualJournal(response.manualJournal) : null;
  }

  async createDraft(values: ManualJournalDraftValues) {
    const response = await requestGraphQL({
      document: CreateManualJournalDraftDocument,
      operationName: "CreateManualJournalDraft",
      variables: { input: toManualJournalDraftInput(values) },
    });
    return toManualJournal(response.createManualJournalDraft);
  }

  async updateDraft(id: string, values: ManualJournalDraftValues) {
    const response = await requestGraphQL({
      document: UpdateManualJournalDraftDocument,
      operationName: "UpdateManualJournalDraft",
      variables: { id, input: toManualJournalDraftInput(values) },
    });
    return toManualJournal(response.updateManualJournalDraft);
  }

  async submit(id: string) {
    const response = await requestGraphQL({
      document: SubmitManualJournalDocument,
      operationName: "SubmitManualJournal",
      variables: { id },
    });
    return toManualJournal(response.submitManualJournal);
  }

  async approve(id: string) {
    const response = await requestGraphQL({
      document: ApproveManualJournalDocument,
      operationName: "ApproveManualJournal",
      variables: { id },
    });
    return toManualJournal(response.approveManualJournal);
  }

  async post(id: string) {
    const response = await requestGraphQL({
      document: PostManualJournalDocument,
      operationName: "PostManualJournal",
      variables: { id },
    });
    return toManualJournal(response.postManualJournal);
  }

  async reject(id: string, reason: string) {
    const response = await requestGraphQL({
      document: RejectManualJournalDocument,
      operationName: "RejectManualJournal",
      variables: { id, reason },
    });
    return toManualJournal(response.rejectManualJournal);
  }

  async cancel(id: string, reason: string) {
    const response = await requestGraphQL({
      document: CancelManualJournalDocument,
      operationName: "CancelManualJournal",
      variables: { id, reason },
    });
    return toManualJournal(response.cancelManualJournal);
  }
}
