import {
  ActivateFiscalYearDocument,
  CloseFiscalYearDocument,
  CreateFiscalYearDocument,
  FiscalYearClosePreviewDocument,
  ReopenFiscalYearDocument,
  UpdateFiscalYearDocument,
  type FiscalYearInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { FiscalYear, FiscalYearClosePlan } from "@/types/fiscal-year";

type RequestOptions = { signal?: AbortSignal };

export function toFiscalYearInput(data: FiscalYear): FiscalYearInput {
  return {
    status: data.status,
    year: data.year,
    name: data.name,
    description: data.description ?? null,
    startDate: data.startDate,
    endDate: data.endDate,
    isCurrent: data.isCurrent ?? false,
    isCalendarYear: data.isCalendarYear ?? false,
    allowAdjustingEntries: data.allowAdjustingEntries ?? false,
    version: data.version ?? null,
  };
}

export class FiscalYearService {
  async create(data: FiscalYear): Promise<FiscalYear> {
    const response = await requestGraphQL({
      document: CreateFiscalYearDocument,
      operationName: "CreateFiscalYear",
      variables: { input: toFiscalYearInput(data) },
    });
    return response.createFiscalYear;
  }

  async update(id: string, data: FiscalYear): Promise<FiscalYear> {
    const response = await requestGraphQL({
      document: UpdateFiscalYearDocument,
      operationName: "UpdateFiscalYear",
      variables: { id, input: toFiscalYearInput(data) },
    });
    return response.updateFiscalYear;
  }

  async close(id: FiscalYear["id"]): Promise<FiscalYear> {
    const response = await requestGraphQL({
      document: CloseFiscalYearDocument,
      operationName: "CloseFiscalYear",
      variables: { id: id ?? "" },
    });
    return response.closeFiscalYear;
  }

  async closePreview(
    id: FiscalYear["id"],
    { signal }: RequestOptions = {},
  ): Promise<FiscalYearClosePlan> {
    const response = await requestGraphQL({
      document: FiscalYearClosePreviewDocument,
      operationName: "FiscalYearClosePreview",
      variables: { id: id ?? "" },
      signal,
    });
    return response.fiscalYearClosePreview;
  }

  async reopen(id: FiscalYear["id"], reopenReason: string): Promise<FiscalYear> {
    const response = await requestGraphQL({
      document: ReopenFiscalYearDocument,
      operationName: "ReopenFiscalYear",
      variables: { id: id ?? "", reopenReason },
    });
    return response.reopenFiscalYear;
  }

  async activate(id: FiscalYear["id"]): Promise<FiscalYear> {
    const response = await requestGraphQL({
      document: ActivateFiscalYearDocument,
      operationName: "ActivateFiscalYear",
      variables: { id: id ?? "" },
    });
    return response.activateFiscalYear;
  }
}
