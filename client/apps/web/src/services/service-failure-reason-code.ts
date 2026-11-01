import {
  ActivateServiceFailureReasonCodeDocument,
  ArchiveServiceFailureReasonCodeDocument,
  CreateServiceFailureReasonCodeDocument,
  UpdateServiceFailureReasonCodeDocument,
  type ServiceFailureReasonCodeInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";

export function toServiceFailureReasonCodeInput(
  data: ServiceFailureReasonCode,
): ServiceFailureReasonCodeInput {
  return {
    code: data.code,
    label: data.label,
    description: data.description || null,
    category: data.category,
    appliesTo: data.appliesTo,
    defaultStatusCode: data.defaultStatusCode || null,
    defaultReasonCode: data.defaultReasonCode || null,
    defaultExceptionCode: data.defaultExceptionCode || null,
    defaultNote: data.defaultNote || null,
    active: data.active,
    sortOrder: data.sortOrder,
    externalMap: data.externalMap ?? null,
    version: data.version,
  };
}

export class ServiceFailureReasonCodeService {
  async create(data: ServiceFailureReasonCode) {
    const response = await requestGraphQL({
      document: CreateServiceFailureReasonCodeDocument,
      operationName: "CreateServiceFailureReasonCode",
      variables: { input: toServiceFailureReasonCodeInput(data) },
    });

    return response.createServiceFailureReasonCode;
  }

  async update(id: string, data: ServiceFailureReasonCode) {
    const response = await requestGraphQL({
      document: UpdateServiceFailureReasonCodeDocument,
      operationName: "UpdateServiceFailureReasonCode",
      variables: { id, input: toServiceFailureReasonCodeInput(data) },
    });

    return response.updateServiceFailureReasonCode;
  }

  async archive(id: string) {
    const response = await requestGraphQL({
      document: ArchiveServiceFailureReasonCodeDocument,
      operationName: "ArchiveServiceFailureReasonCode",
      variables: { id },
    });

    return response.archiveServiceFailureReasonCode;
  }

  async activate(id: string) {
    const response = await requestGraphQL({
      document: ActivateServiceFailureReasonCodeDocument,
      operationName: "ActivateServiceFailureReasonCode",
      variables: { id },
    });

    return response.activateServiceFailureReasonCode;
  }
}
