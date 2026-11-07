import {
  AuditAlertPoliciesDocument,
  AuditChainStatusDocument,
  AuditExportDownloadDocument,
  AuditExportTableDocument,
  AuditLegalHoldTableDocument,
  PlaceAuditLegalHoldDocument,
  ReleaseAuditLegalHoldDocument,
  RequestAuditExportDocument,
  UpdateAuditAlertPolicyDocument,
  type AuditAlertKind,
  type AuditAlertPoliciesQuery,
  type AuditChainStatusQuery,
  type AuditExportDownloadMutation,
  type AuditExportFormat,
  type PlaceAuditLegalHoldInput,
  type RequestAuditExportInput,
  type UpdateAuditAlertPolicyInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableConfigRow } from "@trenova/shared/types/data-table";

export type {
  AuditAlertKind,
  AuditExportFormat,
  PlaceAuditLegalHoldInput,
  RequestAuditExportInput,
  UpdateAuditAlertPolicyInput,
};

type RequestOptions = { signal?: AbortSignal };

export const AUDIT_LEGAL_HOLD_LIST_KEY = "audit-legal-hold-list";
export const AUDIT_EXPORT_LIST_KEY = "audit-export-list";

export const auditLegalHoldTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AuditLegalHoldTableDocument,
  operationName: "AuditLegalHoldTable",
  connectionKey: "auditLegalHolds",
});

export type AuditLegalHoldRow = DataTableConfigRow<typeof auditLegalHoldTableGraphQLConfig>;

export const auditExportTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: AuditExportTableDocument,
  operationName: "AuditExportTable",
  connectionKey: "auditExports",
});

export type AuditExportRow = DataTableConfigRow<typeof auditExportTableGraphQLConfig>;

export type AuditChainStatus = AuditChainStatusQuery["auditChainStatus"];
export type AuditAlertPolicy = AuditAlertPoliciesQuery["auditAlertPolicies"][number];
export type AuditExportDownload = AuditExportDownloadMutation["auditExportDownload"];

export async function fetchAuditChainStatus(options?: RequestOptions): Promise<AuditChainStatus> {
  const data = await requestGraphQL({
    document: AuditChainStatusDocument,
    operationName: "AuditChainStatus",
    variables: {},
    signal: options?.signal,
  });
  return data.auditChainStatus;
}

export async function fetchAuditAlertPolicies(
  options?: RequestOptions,
): Promise<AuditAlertPolicy[]> {
  const data = await requestGraphQL({
    document: AuditAlertPoliciesDocument,
    operationName: "AuditAlertPolicies",
    variables: {},
    signal: options?.signal,
  });
  return data.auditAlertPolicies;
}

export async function updateAuditAlertPolicy(
  input: UpdateAuditAlertPolicyInput,
): Promise<AuditAlertPolicy> {
  const data = await requestGraphQL({
    document: UpdateAuditAlertPolicyDocument,
    operationName: "UpdateAuditAlertPolicy",
    variables: { input },
  });
  return data.updateAuditAlertPolicy;
}

export async function placeAuditLegalHold(input: PlaceAuditLegalHoldInput) {
  const data = await requestGraphQL({
    document: PlaceAuditLegalHoldDocument,
    operationName: "PlaceAuditLegalHold",
    variables: { input },
  });
  return data.placeAuditLegalHold;
}

export async function releaseAuditLegalHold(id: string, reason: string) {
  const data = await requestGraphQL({
    document: ReleaseAuditLegalHoldDocument,
    operationName: "ReleaseAuditLegalHold",
    variables: { id, reason },
  });
  return data.releaseAuditLegalHold;
}

export async function requestAuditExport(input: RequestAuditExportInput) {
  const data = await requestGraphQL({
    document: RequestAuditExportDocument,
    operationName: "RequestAuditExport",
    variables: { input },
  });
  return data.requestAuditExport;
}

export async function fetchAuditExportDownload(id: string): Promise<AuditExportDownload> {
  const data = await requestGraphQL({
    document: AuditExportDownloadDocument,
    operationName: "AuditExportDownload",
    variables: { id },
  });
  return data.auditExportDownload;
}
