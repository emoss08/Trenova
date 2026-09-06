import {
  ClockInDocument,
  ClockOutDocument,
  DeleteTimeEntryDocument,
  GeneratePayrollExportDocument,
  OpenTimeClockEntryDocument,
  PayrollExportRowsDocument,
  PayrollExportsDocument,
  RecordTimeEntryDocument,
  TimeClockEntriesDocument,
  TimesheetDocument,
  TimesheetsDocument,
  TransitionTimesheetDocument,
  VoidPayrollExportDocument,
  type ClockInMutation,
  type ClockInMutationVariables,
  type ClockInput,
  type ClockOutMutation,
  type ClockOutMutationVariables,
  type DeleteTimeEntryInput,
  type DeleteTimeEntryMutation,
  type DeleteTimeEntryMutationVariables,
  type GeneratePayrollExportInput,
  type GeneratePayrollExportMutation,
  type GeneratePayrollExportMutationVariables,
  type OpenTimeClockEntryQuery,
  type OpenTimeClockEntryQueryVariables,
  type PayrollExportRowsQuery,
  type PayrollExportRowsQueryVariables,
  type PayrollExportsQuery,
  type PayrollExportsQueryVariables,
  type RecordTimeEntryInput,
  type RecordTimeEntryMutation,
  type RecordTimeEntryMutationVariables,
  type TimeClockEntriesQuery,
  type TimeClockEntriesQueryVariables,
  type TimesheetFilterInput,
  type TimesheetQuery,
  type TimesheetQueryVariables,
  type TimesheetsQuery,
  type TimesheetsQueryVariables,
  type TransitionTimesheetInput,
  type TransitionTimesheetMutation,
  type TransitionTimesheetMutationVariables,
  type VoidPayrollExportInput,
  type VoidPayrollExportMutation,
  type VoidPayrollExportMutationVariables,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type TimesheetRow = TimesheetsQuery["timesheets"][number];
export type TimesheetDetail = TimesheetQuery["timesheet"];
export type TimeEntryRow = NonNullable<TimesheetDetail["entries"]>[number];
export type OpenTimeEntry = OpenTimeClockEntryQuery["openTimeClockEntry"];
export type PayrollExportRow = PayrollExportsQuery["payrollExports"][number];
export type PayrollExportLine = PayrollExportRowsQuery["payrollExportRows"][number];

export const TIMESHEETS_KEY = "timesheets";
export const TIMESHEET_KEY = "timesheet";
export const TIME_ENTRIES_KEY = "time-clock-entries";
export const OPEN_ENTRY_KEY = "open-time-clock-entry";
export const PAYROLL_EXPORTS_KEY = "payroll-exports";

export async function fetchTimesheets(
  filter: TimesheetFilterInput = {},
  options?: { signal?: AbortSignal },
): Promise<TimesheetRow[]> {
  const data = await requestGraphQL<TimesheetsQuery, TimesheetsQueryVariables>({
    document: TimesheetsDocument,
    operationName: "Timesheets",
    variables: { filter },
    signal: options?.signal,
  });
  return data.timesheets;
}

export async function fetchTimesheet(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<TimesheetDetail> {
  const data = await requestGraphQL<TimesheetQuery, TimesheetQueryVariables>({
    document: TimesheetDocument,
    operationName: "Timesheet",
    variables: { id },
    signal: options?.signal,
  });
  return data.timesheet;
}

export async function fetchOpenTimeEntry(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<OpenTimeEntry> {
  const data = await requestGraphQL<OpenTimeClockEntryQuery, OpenTimeClockEntryQueryVariables>({
    document: OpenTimeClockEntryDocument,
    operationName: "OpenTimeClockEntry",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.openTimeClockEntry;
}

export async function fetchTimeClockEntries(
  args: {
    workerId: string;
    from?: number;
    to?: number;
    limit?: number;
  },
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL<TimeClockEntriesQuery, TimeClockEntriesQueryVariables>({
    document: TimeClockEntriesDocument,
    operationName: "TimeClockEntries",
    variables: {
      workerId: args.workerId,
      from: args.from ?? null,
      to: args.to ?? null,
      limit: args.limit ?? null,
    },
    signal: options?.signal,
  });
  return data.timeClockEntries;
}

export async function fetchPayrollExports(
  limit?: number,
  options?: { signal?: AbortSignal },
): Promise<PayrollExportRow[]> {
  const data = await requestGraphQL<PayrollExportsQuery, PayrollExportsQueryVariables>({
    document: PayrollExportsDocument,
    operationName: "PayrollExports",
    variables: { limit: limit ?? null },
    signal: options?.signal,
  });
  return data.payrollExports;
}

export async function fetchPayrollExportRows(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<PayrollExportLine[]> {
  const data = await requestGraphQL<PayrollExportRowsQuery, PayrollExportRowsQueryVariables>({
    document: PayrollExportRowsDocument,
    operationName: "PayrollExportRows",
    variables: { id },
    signal: options?.signal,
  });
  return data.payrollExportRows;
}

export async function clockIn(input: ClockInput) {
  const data = await requestGraphQL<ClockInMutation, ClockInMutationVariables>({
    document: ClockInDocument,
    operationName: "ClockIn",
    variables: { input },
  });
  return data.clockIn;
}

export async function clockOut(input: ClockInput) {
  const data = await requestGraphQL<ClockOutMutation, ClockOutMutationVariables>({
    document: ClockOutDocument,
    operationName: "ClockOut",
    variables: { input },
  });
  return data.clockOut;
}

export async function recordTimeEntry(input: RecordTimeEntryInput) {
  const data = await requestGraphQL<RecordTimeEntryMutation, RecordTimeEntryMutationVariables>({
    document: RecordTimeEntryDocument,
    operationName: "RecordTimeEntry",
    variables: { input },
  });
  return data.recordTimeEntry;
}

export async function deleteTimeEntry(input: DeleteTimeEntryInput) {
  const data = await requestGraphQL<DeleteTimeEntryMutation, DeleteTimeEntryMutationVariables>({
    document: DeleteTimeEntryDocument,
    operationName: "DeleteTimeEntry",
    variables: { input },
  });
  return data.deleteTimeEntry;
}

export async function transitionTimesheet(input: TransitionTimesheetInput) {
  const data = await requestGraphQL<
    TransitionTimesheetMutation,
    TransitionTimesheetMutationVariables
  >({
    document: TransitionTimesheetDocument,
    operationName: "TransitionTimesheet",
    variables: { input },
  });
  return data.transitionTimesheet;
}

export async function generatePayrollExport(input: GeneratePayrollExportInput) {
  const data = await requestGraphQL<
    GeneratePayrollExportMutation,
    GeneratePayrollExportMutationVariables
  >({
    document: GeneratePayrollExportDocument,
    operationName: "GeneratePayrollExport",
    variables: { input },
  });
  return data.generatePayrollExport;
}

export async function voidPayrollExport(input: VoidPayrollExportInput) {
  const data = await requestGraphQL<VoidPayrollExportMutation, VoidPayrollExportMutationVariables>({
    document: VoidPayrollExportDocument,
    operationName: "VoidPayrollExport",
    variables: { input },
  });
  return data.voidPayrollExport;
}
