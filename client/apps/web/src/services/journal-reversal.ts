import {
  ApproveJournalReversalDocument,
  CancelJournalReversalDocument,
  CreateJournalReversalDocument,
  PostJournalReversalDocument,
  RejectJournalReversalDocument,
  type CreateJournalReversalInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export class JournalReversalService {
  async create(input: CreateJournalReversalInput) {
    const response = await requestGraphQL({
      document: CreateJournalReversalDocument,
      operationName: "CreateJournalReversal",
      variables: { input },
    });
    return response.createJournalReversal;
  }

  async approve(id: string) {
    const response = await requestGraphQL({
      document: ApproveJournalReversalDocument,
      operationName: "ApproveJournalReversal",
      variables: { id },
    });
    return response.approveJournalReversal;
  }

  async post(id: string) {
    const response = await requestGraphQL({
      document: PostJournalReversalDocument,
      operationName: "PostJournalReversal",
      variables: { id },
    });
    return response.postJournalReversal;
  }

  async reject(id: string, reason: string) {
    const response = await requestGraphQL({
      document: RejectJournalReversalDocument,
      operationName: "RejectJournalReversal",
      variables: { id, reason },
    });
    return response.rejectJournalReversal;
  }

  async cancel(id: string, reason: string) {
    const response = await requestGraphQL({
      document: CancelJournalReversalDocument,
      operationName: "CancelJournalReversal",
      variables: { id, reason },
    });
    return response.cancelJournalReversal;
  }
}
