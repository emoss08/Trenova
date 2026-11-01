import {
  SendCustomerStatementDocument,
  SentCustomerStatementsDocument,
} from "@trenova/graphql/generated/graphql";
import { api } from "@trenova/shared/lib/api";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type {
  SentStatement,
  StatementRange,
  StatementSendResult,
} from "@/types/customer-statement";

type RequestOptions = { signal?: AbortSignal };

const customerPath = (customerId: string) =>
  `/accounting/accounts-receivable/customers/${encodeURIComponent(customerId)}`;

function rangeQuery(range: StatementRange): string {
  const query = new URLSearchParams();
  if (range.startDate) query.set("startDate", String(range.startDate));
  if (range.asOfDate) query.set("asOfDate", String(range.asOfDate));
  const text = query.toString();
  return text ? `?${text}` : "";
}

export class CustomerStatementService {
  /** The statement as the PDF the customer receives. */
  async downloadPdf(customerId: string, range: StatementRange) {
    return api.download(
      `${customerPath(customerId)}/statement/pdf/${rangeQuery(range)}`,
      "Statement.pdf",
    );
  }

  /** Emails the statement to the customer's invoice recipients, PDF attached. */
  async send(
    customerId: string,
    range: StatementRange,
    note: string,
  ): Promise<StatementSendResult> {
    const response = await requestGraphQL({
      document: SendCustomerStatementDocument,
      operationName: "SendCustomerStatement",
      variables: {
        customerId,
        input: {
          startDate: range.startDate || null,
          asOfDate: range.asOfDate || null,
          note,
        },
      },
    });
    return response.sendCustomerStatement;
  }

  /** The statements the customer has been sent, newest first. */
  async listSent(customerId: string, { signal }: RequestOptions = {}): Promise<SentStatement[]> {
    const response = await requestGraphQL({
      document: SentCustomerStatementsDocument,
      operationName: "SentCustomerStatements",
      variables: { customerId },
      signal,
    });
    return response.sentCustomerStatements;
  }
}
