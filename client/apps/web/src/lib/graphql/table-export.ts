import type {
  ExportTableViewInput,
  ScheduleTableViewInput,
} from "@trenova/graphql/generated/graphql";
import {
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
