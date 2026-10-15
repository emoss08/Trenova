import {
  ApplyFactoringRemittanceDocument,
  CreateFactoringRemittanceDocument,
  DiscardFactoringRemittanceDocument,
  FactoringPlannedMovementFieldsFragmentDoc,
  FactoringRemittanceDetailDocument,
  FactoringRemittanceFieldsFragmentDoc,
  FactoringRemittanceLineFieldsFragmentDoc,
  FactoringRemittanceLinesDocument,
  FactoringRemittancePlanDocument,
  FactoringRemittanceTableDocument,
  FactoringSubmissionTableDocument,
  FactoringRemittancesDocument,
  RematchFactoringRemittanceDocument,
  StageFactoringRemittanceDocument,
  UpdateFactoringRemittanceLineDocument,
  type CreateFactoringRemittanceInput,
  type DiscardFactoringRemittanceInput,
  type FactoringRemittanceFieldsFragment,
  type FactoringRemittanceLineFieldsFragment,
  type FactoringRemittanceMatchStatus,
  type FactoringRemittancePlanQuery,
  type FactoringRemittanceStatus,
  type FactoringRemittanceVersionInput,
  type StageFactoringRemittanceInput,
  type UpdateFactoringRemittanceLineInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";
import type { FactoringPlannedMovement } from "@/lib/graphql/factoring";

type RequestOptions = { signal?: AbortSignal };

export type FactoringRemittance = FactoringRemittanceFieldsFragment;
export type FactoringRemittanceLine = FactoringRemittanceLineFieldsFragment;
export type FactoringRemittanceSuggestion = FactoringRemittanceLine["suggestions"][number];
type RawRemittancePlan = FactoringRemittancePlanQuery["factoringRemittancePlan"];
export type FactoringRemittancePlanLine = Omit<RawRemittancePlan["lines"][number], "movement"> & {
  movement: FactoringPlannedMovement | null;
};
export type FactoringRemittancePlan = Omit<RawRemittancePlan, "lines"> & {
  lines: FactoringRemittancePlanLine[];
};

/** The document resource a remittance's file is uploaded under. */
export const FACTORING_REMITTANCE_RESOURCE = "factoring_remittance";

/** The most remittances one listing reads; a factor sends a handful a week. */
export const FACTORING_REMITTANCE_PAGE_SIZE = 50;

/** A remittance holds at most this many lines, so one page reads all of them. */
export const FACTORING_REMITTANCE_LINE_LIMIT = 2000;

export function createFactoringRemittanceTableConfig(factoringCompanyId: string | null) {
  return defineDataTableGraphQLConfig({
    document: FactoringRemittanceTableDocument,
    operationName: "FactoringRemittanceTable",
    connectionKey: "factoringRemittances",
    extraVariables: { factoringCompanyId },
  });
}

export type FactoringRemittanceRow = DataTableConfigRow<
  ReturnType<typeof createFactoringRemittanceTableConfig>
>;

export type FactoringRemittanceFilter = {
  factoringCompanyId?: string;
  inboundMessageId?: string;
  statuses?: FactoringRemittanceStatus[];
};

export type FactoringRemittancePage = {
  remittances: FactoringRemittance[];
  totalCount: number;
};

export async function listFactoringRemittances(
  filter: FactoringRemittanceFilter,
  options?: RequestOptions,
): Promise<FactoringRemittancePage> {
  const data = await requestGraphQL({
    document: FactoringRemittancesDocument,
    operationName: "FactoringRemittances",
    variables: {
      input: {
        table: {
          first: FACTORING_REMITTANCE_PAGE_SIZE,
          sort: [{ field: "createdAt", direction: "desc" }],
        },
        factoringCompanyId: filter.factoringCompanyId ?? null,
        inboundMessageId: filter.inboundMessageId ?? null,
        statuses: filter.statuses ?? null,
      },
    },
    signal: options?.signal,
  });
  const remittances = data.factoringRemittances.edges.map((edge) =>
    getFragmentData(FactoringRemittanceFieldsFragmentDoc, edge.node),
  );
  return {
    remittances,
    totalCount: data.factoringRemittances.totalCount ?? remittances.length,
  };
}

export async function fetchFactoringRemittance(
  id: string,
  options?: RequestOptions,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: FactoringRemittanceDetailDocument,
    operationName: "FactoringRemittanceDetail",
    variables: { id },
    signal: options?.signal,
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.factoringRemittance);
}

export async function listFactoringRemittanceLines(
  remittanceId: string,
  matchStatuses: FactoringRemittanceMatchStatus[] | null,
  options?: RequestOptions,
): Promise<FactoringRemittanceLine[]> {
  const data = await requestGraphQL({
    document: FactoringRemittanceLinesDocument,
    operationName: "FactoringRemittanceLines",
    variables: {
      input: {
        table: {
          first: FACTORING_REMITTANCE_LINE_LIMIT,
          sort: [{ field: "rowNumber", direction: "asc" }],
        },
        remittanceId,
        matchStatuses,
      },
    },
    signal: options?.signal,
  });
  return data.factoringRemittanceLines.edges.map((edge) =>
    getFragmentData(FactoringRemittanceLineFieldsFragmentDoc, edge.node),
  );
}

export async function fetchFactoringRemittancePlan(
  input: FactoringRemittanceVersionInput,
  options?: RequestOptions,
): Promise<FactoringRemittancePlan> {
  const data = await requestGraphQL({
    document: FactoringRemittancePlanDocument,
    operationName: "FactoringRemittancePlan",
    variables: { input },
    signal: options?.signal,
  });
  const { lines, ...plan } = data.factoringRemittancePlan;
  return {
    ...plan,
    lines: lines.map(({ movement, ...line }) => ({
      ...line,
      movement: movement
        ? getFragmentData(FactoringPlannedMovementFieldsFragmentDoc, movement)
        : null,
    })),
  };
}

export async function createFactoringRemittance(
  input: CreateFactoringRemittanceInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: CreateFactoringRemittanceDocument,
    operationName: "CreateFactoringRemittance",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.createFactoringRemittance);
}

export async function stageFactoringRemittance(
  input: StageFactoringRemittanceInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: StageFactoringRemittanceDocument,
    operationName: "StageFactoringRemittance",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.stageFactoringRemittance);
}

export async function rematchFactoringRemittance(
  input: FactoringRemittanceVersionInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: RematchFactoringRemittanceDocument,
    operationName: "RematchFactoringRemittance",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.rematchFactoringRemittance);
}

export async function updateFactoringRemittanceLine(
  input: UpdateFactoringRemittanceLineInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: UpdateFactoringRemittanceLineDocument,
    operationName: "UpdateFactoringRemittanceLine",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.updateFactoringRemittanceLine);
}

export async function applyFactoringRemittance(
  input: FactoringRemittanceVersionInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: ApplyFactoringRemittanceDocument,
    operationName: "ApplyFactoringRemittance",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.applyFactoringRemittance);
}

export async function discardFactoringRemittance(
  input: DiscardFactoringRemittanceInput,
): Promise<FactoringRemittance> {
  const data = await requestGraphQL({
    document: DiscardFactoringRemittanceDocument,
    operationName: "DiscardFactoringRemittance",
    variables: { input },
  });
  return getFragmentData(FactoringRemittanceFieldsFragmentDoc, data.discardFactoringRemittance);
}

export type OpenFactoringSchedule = {
  id: string;
  number: string;
  status: string;
  invoiceCount: number;
};

/** The factor's schedules a remittance line can still pay: sent, partly funded or funded. */
export async function listOpenFactoringSchedules(
  factoringCompanyId: string,
  options?: RequestOptions,
): Promise<OpenFactoringSchedule[]> {
  const data = await requestGraphQL({
    document: FactoringSubmissionTableDocument,
    operationName: "FactoringSubmissionTable",
    variables: {
      input: {
        first: FACTORING_REMITTANCE_PAGE_SIZE,
        sort: [{ field: "createdAt", direction: "desc" }],
      },
      factoringCompanyId,
      statuses: ["Sent", "PartiallyFunded", "Funded"],
      includeTotalCount: false,
    },
    signal: options?.signal,
  });
  return data.factoringSubmissions.edges.map(({ node }) => ({
    id: node.id,
    number: node.number,
    status: node.status,
    invoiceCount: node.invoiceCount,
  }));
}
