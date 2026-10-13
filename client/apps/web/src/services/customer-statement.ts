import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  sentStatementListSchema,
  statementSendResultSchema,
  type StatementRange,
  type StatementSendResult,
} from "@/types/customer-statement";

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
  async send(customerId: string, range: StatementRange, note: string) {
    const response = await api.post<StatementSendResult>(
      `${customerPath(customerId)}/statement/send/`,
      { ...range, note },
    );
    return safeParse(statementSendResultSchema, response, "StatementSendResult");
  }

  /** The statements the customer has been sent, newest first. */
  async listSent(customerId: string) {
    const response = await api.get(`${customerPath(customerId)}/statements/`);
    const parsed = await safeParse(sentStatementListSchema, response, "SentStatementList");
    return parsed.statements;
  }
}
