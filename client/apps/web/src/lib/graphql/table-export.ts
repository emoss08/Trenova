import type {
  ExportCustomerLedgerInput,
  ExportTableViewInput,
  ScheduleTableViewInput,
} from "@trenova/graphql/generated/graphql";
import {
  ExportCustomerLedgerDocument,
  ExportTableViewDocument,
  ScheduleTableViewDocument,
} from "@trenova/graphql/generated/documents/reports/table-export";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type TableExportOutcome = {
  definitionId: string;
  skippedColumns: string[];
};

export async function exportTableView(input: ExportTableViewInput): Promise<TableExportOutcome> {
  const data = await requestGraphQL({
    document: ExportTableViewDocument,
    operationName: "ExportTableView",
    variables: { input },
  });
  return data.exportTableView;
}

export async function scheduleTableView(
  input: ScheduleTableViewInput,
): Promise<TableExportOutcome> {
  const data = await requestGraphQL({
    document: ScheduleTableViewDocument,
    operationName: "ScheduleTableView",
    variables: { input },
  });
  return data.scheduleTableView;
}

export async function exportCustomerLedger(
  input: ExportCustomerLedgerInput,
): Promise<{ definitionId: string }> {
  const data = await requestGraphQL({
    document: ExportCustomerLedgerDocument,
    operationName: "ExportCustomerLedger",
    variables: { input },
  });
  return data.exportCustomerLedger;
}
