import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  serviceFailureEdi214LifecycleResultSchema,
  serviceFailureEdiPayloadResultSchema,
  serviceFailureEvaluationResultSchema,
  serviceFailureSchema,
  type ServiceFailure,
  type ServiceFailureEdi214LifecycleResult,
  type ServiceFailureEdiPayloadResult,
  type ServiceFailureLifecycleRequest,
  type ServiceFailureUpdate,
} from "@/types/service-failure";
import type { GenericLimitOffsetResponse } from "@/types/server";

export class ServiceFailureService {
  async getAll(params?: string) {
    const response = await api.get<GenericLimitOffsetResponse<ServiceFailure>>(
      `/service-failures/${params ? `?${params}` : ""}`,
    );
    const results = await safeParse(
      serviceFailureSchema.array(),
      response.results,
      "ServiceFailures",
    );

    return {
      ...response,
      results,
    };
  }

  async getById(id: string) {
    const response = await api.get<ServiceFailure>(`/service-failures/${id}/`);
    return safeParse(serviceFailureSchema, response, "ServiceFailure");
  }

  async listByShipment(shipmentId: string, params?: string) {
    const search = new URLSearchParams(params);
    search.set("shipmentId", shipmentId);

    const response = await api.get<GenericLimitOffsetResponse<ServiceFailure>>(
      `/service-failures/?${search.toString()}`,
    );
    const results = await safeParse(
      serviceFailureSchema.array(),
      response.results,
      "ShipmentServiceFailures",
    );

    return {
      ...response,
      results,
    };
  }

  async evaluateShipment(shipmentId: string, force = false) {
    const response = await api.post(
      `/service-failures/evaluate-shipment/${shipmentId}/${
        force ? "?force=true" : ""
      }`,
    );
    return safeParse(
      serviceFailureEvaluationResultSchema,
      response,
      "ServiceFailureEvaluationResult",
    );
  }

  async evaluateStop(params: {
    shipmentId: string;
    shipmentMoveId?: string;
    stopId: string;
    force?: boolean;
  }) {
    const search = new URLSearchParams();
    if (params.shipmentMoveId) search.set("shipmentMoveId", params.shipmentMoveId);
    if (params.force) search.set("force", "true");

    const response = await api.post(
      `/service-failures/evaluate-stop/${params.shipmentId}/${params.stopId}/${
        search.size ? `?${search.toString()}` : ""
      }`,
    );
    return safeParse(
      serviceFailureEvaluationResultSchema,
      response,
      "ServiceFailureEvaluationResult",
    );
  }

  async bulkEvaluate(shipmentIds: string[], force = false) {
    const response = await api.post("/service-failures/bulk-evaluate/", {
      shipmentIds,
      force,
    });
    return safeParse(
      serviceFailureEvaluationResultSchema,
      response,
      "ServiceFailureEvaluationResult",
    );
  }

  async update(id: string, data: ServiceFailureUpdate) {
    const response = await api.put<ServiceFailure>(
      `/service-failures/${id}/`,
      data,
    );
    return safeParse(serviceFailureSchema, response, "ServiceFailure");
  }

  async review(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle(id, "review", data);
  }

  async resolve(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle(id, "resolve", data);
  }

  async void(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle(id, "void", data);
  }

  async buildEDI214Payload(id: string): Promise<ServiceFailureEdiPayloadResult> {
    const response = await api.post(`/service-failures/${id}/edi-214-payload/`);
    return safeParse(
      serviceFailureEdiPayloadResultSchema,
      response,
      "ServiceFailureEDIPayload",
    );
  }

  async edi214Readiness(
    id: string,
    trigger?: "Reviewed" | "Resolved",
  ): Promise<ServiceFailureEdi214LifecycleResult> {
    const search = new URLSearchParams();
    if (trigger) search.set("trigger", trigger);
    const response = await api.get(
      `/service-failures/${id}/edi-214-readiness/${
        search.size ? `?${search.toString()}` : ""
      }`,
    );
    return safeParse(
      serviceFailureEdi214LifecycleResultSchema,
      response,
      "ServiceFailureEDI214Readiness",
    );
  }

  private async lifecycle(
    id: string,
    action: "review" | "resolve" | "void",
    data: ServiceFailureLifecycleRequest,
  ) {
    const response = await api.post<ServiceFailure>(
      `/service-failures/${id}/${action}/`,
      data,
    );
    return safeParse(serviceFailureSchema, response, "ServiceFailure");
  }
}
