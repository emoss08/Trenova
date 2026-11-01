import {
  changeServiceFailureLifecycleGraphQL,
  evaluateShipmentServiceFailuresGraphQL,
  getServiceFailureEdi214PayloadGraphQL,
  getServiceFailureEdi214ReadinessGraphQL,
  getServiceFailureEdi214StatusGraphQL,
  listShipmentServiceFailuresGraphQL,
  updateServiceFailureGraphQL,
  type ServiceFailureLifecycleAction,
} from "@/lib/graphql/service-failure";
import type { ServiceFailureRow } from "@/lib/graphql/service-failure-table";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  serviceFailureEdi214LifecycleResultSchema,
  serviceFailureEdi214StatusSchema,
  serviceFailureEdiPayloadResultSchema,
  serviceFailureEvaluationResultSchema,
  type ServiceFailureEdi214LifecycleResult,
  type ServiceFailureEdi214Status,
  type ServiceFailureEdiPayloadResult,
  type ServiceFailureLifecycleRequest,
  type ServiceFailureUpdate,
} from "@/types/service-failure";

/** GraphQL sends an unset part of a stop summary as an empty string; the summary leaves it out. */
function withoutEmptyParts<T extends Record<string, unknown>>(stop: T): Partial<T> {
  return Object.fromEntries(
    Object.entries(stop).filter(([, value]) => value !== "" && value !== null),
  ) as Partial<T>;
}

const SHIPMENT_FAILURE_LIMIT = 100;

export class ServiceFailureService {
  async listByShipment(shipmentId: string, { signal }: { signal?: AbortSignal } = {}) {
    return listShipmentServiceFailuresGraphQL(shipmentId, SHIPMENT_FAILURE_LIMIT, { signal });
  }

  async evaluateShipment(shipmentId: string, force = false) {
    const result = await evaluateShipmentServiceFailuresGraphQL(shipmentId, force);
    return safeParse(
      serviceFailureEvaluationResultSchema,
      {
        ...result,
        createdStops: result.createdStops.map(withoutEmptyParts),
        updatedStops: result.updatedStops.map(withoutEmptyParts),
        skippedStops: result.skippedStops.map(withoutEmptyParts),
      },
      "ServiceFailureEvaluationResult",
    );
  }

  async update(id: string, data: ServiceFailureUpdate): Promise<ServiceFailureRow> {
    return updateServiceFailureGraphQL(id, {
      reasonCodeId: data.reasonCodeId || null,
      clearReasonCode: data.clearReasonCode,
      notes: data.notes,
      internalNotes: data.internalNotes,
      x12StatusCodeOverride: data.x12StatusCodeOverride,
      x12ReasonCodeOverride: data.x12ReasonCodeOverride,
      x12ExceptionCode: data.x12ExceptionCode,
      version: data.version,
    });
  }

  async review(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle("review", id, data);
  }

  async resolve(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle("resolve", id, data);
  }

  async void(id: string, data: ServiceFailureLifecycleRequest) {
    return this.lifecycle("void", id, data);
  }

  async buildEDI214Payload(id: string): Promise<ServiceFailureEdiPayloadResult> {
    const response = await getServiceFailureEdi214PayloadGraphQL(id);
    return safeParse(serviceFailureEdiPayloadResultSchema, response, "ServiceFailureEDIPayload");
  }

  async edi214Readiness(
    id: string,
    trigger?: "Reviewed" | "Resolved",
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<ServiceFailureEdi214LifecycleResult> {
    const response = await getServiceFailureEdi214ReadinessGraphQL(id, trigger, { signal });
    return safeParse(
      serviceFailureEdi214LifecycleResultSchema,
      response,
      "ServiceFailureEDI214Readiness",
    );
  }

  async edi214Status(
    id: string,
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<ServiceFailureEdi214Status> {
    const response = await getServiceFailureEdi214StatusGraphQL(id, { signal });
    return safeParse(serviceFailureEdi214StatusSchema, response, "ServiceFailureEDI214Status");
  }

  private async lifecycle(
    action: ServiceFailureLifecycleAction,
    id: string,
    data: ServiceFailureLifecycleRequest,
  ): Promise<ServiceFailureRow> {
    return changeServiceFailureLifecycleGraphQL(action, id, {
      reasonCodeId: data.reasonCodeId || null,
      notes: data.notes ?? null,
      version: data.version,
    });
  }
}
