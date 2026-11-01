import {
  assignDispatchMoveToCarrierGraphQL,
  cancelDispatchCarrierAssignmentGraphQL,
  getDispatchCarrierAssignmentPreviewGraphQL,
} from "@/lib/graphql/dispatch-console";
import type { DispatchAssignMoveToCarrierInput } from "@trenova/graphql/generated/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  carrierAssignmentPayloadSchema,
  carrierAssignmentSchema,
  carrierEligibilitySchema,
  type CarrierAssignment,
  type CarrierAssignmentPayload,
  type CarrierEligibility,
} from "@trenova/shared/types/shipment";

export function toCarrierAssignmentInput(
  moveId: string,
  payload: CarrierAssignmentPayload,
  replace: boolean,
): DispatchAssignMoveToCarrierInput {
  return {
    moveId,
    carrierId: payload.carrierId,
    rateMethod: payload.rateMethod,
    baseRate: payload.baseRate.toString(),
    fuelSurcharge: payload.fuelSurcharge != null ? payload.fuelSurcharge.toString() : null,
    accessorials: payload.accessorials.map((accessorial) => ({
      accessorialChargeId: accessorial.accessorialChargeId ?? null,
      description: accessorial.description,
      amount: accessorial.amount.toString(),
    })),
    proNumber: payload.proNumber,
    externalDriverName: payload.externalDriverName,
    externalDriverPhone: payload.externalDriverPhone,
    externalTractorNumber: payload.externalTractorNumber,
    externalTrailerNumber: payload.externalTrailerNumber,
    replace,
    overrideInsuranceWarning: payload.overrideInsuranceWarning,
    autoRate: payload.autoRate,
    overrideMarginFloor: payload.overrideMarginFloor,
  };
}

export class CarrierAssignmentService {
  public async preview(
    carrierId: string,
    { signal }: { signal?: AbortSignal } = {},
  ): Promise<CarrierEligibility> {
    const response = await getDispatchCarrierAssignmentPreviewGraphQL({ carrierId }, { signal });

    return safeParse(carrierEligibilitySchema, response, "CarrierEligibility");
  }

  public async assign(
    moveId: string,
    payload: CarrierAssignmentPayload,
    options?: { replace?: boolean },
  ): Promise<CarrierAssignment> {
    const response = await assignDispatchMoveToCarrierGraphQL(
      toCarrierAssignmentInput(
        moveId,
        carrierAssignmentPayloadSchema.parse(payload),
        options?.replace ?? false,
      ),
    );

    return safeParse(carrierAssignmentSchema, response, "CarrierAssignment");
  }

  public async cancel(moveId: string, reason: string): Promise<void> {
    await cancelDispatchCarrierAssignmentGraphQL(moveId, reason);
  }
}
