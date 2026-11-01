import {
  CreateHoldReasonDocument,
  UpdateHoldReasonDocument,
  type HoldReasonInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { HoldReason } from "@/types/hold-reason";

export function toHoldReasonInput(data: HoldReason): HoldReasonInput {
  return {
    type: data.type,
    code: data.code,
    label: data.label,
    description: data.description || null,
    active: data.active,
    defaultSeverity: data.defaultSeverity,
    defaultBlocksDispatch: data.defaultBlocksDispatch,
    defaultBlocksDelivery: data.defaultBlocksDelivery,
    defaultBlocksBilling: data.defaultBlocksBilling,
    defaultVisibleToCustomer: data.defaultVisibleToCustomer,
    sortOrder: data.sortOrder,
    externalMap: data.externalMap ?? null,
    version: data.version,
  };
}

export class HoldReasonService {
  async create(data: HoldReason) {
    const response = await requestGraphQL({
      document: CreateHoldReasonDocument,
      operationName: "CreateHoldReason",
      variables: { input: toHoldReasonInput(data) },
    });

    return response.createHoldReason;
  }

  async update(id: string, data: HoldReason) {
    const response = await requestGraphQL({
      document: UpdateHoldReasonDocument,
      operationName: "UpdateHoldReason",
      variables: { id, input: toHoldReasonInput(data) },
    });

    return response.updateHoldReason;
  }
}
