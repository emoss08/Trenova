import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  serviceFailureReasonCodeSchema,
  type ServiceFailureReasonCode,
} from "@/types/service-failure-reason-code";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";

export class ServiceFailureReasonCodeService {
  async getAll(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<ServiceFailureReasonCode>>(
      `/service-failure-reason-codes/${params ? `?${params}` : ""}`,
    );

    const results = await safeParse(
      serviceFailureReasonCodeSchema.array(),
      response.results,
      "ServiceFailureReasonCodes",
    );

    return {
      ...response,
      results,
    };
  }

  async getById(id: string) {
    const response = await api.get<ServiceFailureReasonCode>(
      `/service-failure-reason-codes/${id}/`,
    );
    return safeParse(serviceFailureReasonCodeSchema, response, "ServiceFailureReasonCode");
  }

  async create(data: ServiceFailureReasonCode) {
    const response = await api.post<ServiceFailureReasonCode>(
      "/service-failure-reason-codes/",
      data,
    );
    return safeParse(serviceFailureReasonCodeSchema, response, "ServiceFailureReasonCode");
  }

  async update(id: string, data: ServiceFailureReasonCode) {
    const response = await api.put<ServiceFailureReasonCode>(
      `/service-failure-reason-codes/${id}/`,
      data,
    );
    return safeParse(serviceFailureReasonCodeSchema, response, "ServiceFailureReasonCode");
  }

  async archive(id: string) {
    const response = await api.post<ServiceFailureReasonCode>(
      `/service-failure-reason-codes/${id}/archive/`,
    );
    return safeParse(serviceFailureReasonCodeSchema, response, "ServiceFailureReasonCode");
  }

  async activate(id: string) {
    const response = await api.post<ServiceFailureReasonCode>(
      `/service-failure-reason-codes/${id}/activate/`,
    );
    return safeParse(serviceFailureReasonCodeSchema, response, "ServiceFailureReasonCode");
  }

  async reorder(reasonIds: string[]) {
    const response = await api.post<ServiceFailureReasonCode[]>(
      "/service-failure-reason-codes/reorder/",
      { reasonIds },
    );
    return safeParse(serviceFailureReasonCodeSchema.array(), response, "ServiceFailureReasonCodes");
  }
}
