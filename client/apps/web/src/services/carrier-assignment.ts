import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  carrierAssignmentPayloadSchema,
  carrierAssignmentSchema,
  carrierEligibilitySchema,
  type CarrierAssignment,
  type CarrierAssignmentPayload,
  type CarrierEligibility,
} from "@trenova/shared/types/shipment";

type CarrierAssignmentRequestBody = {
  carrierId: string;
  rateMethod: CarrierAssignmentPayload["rateMethod"];
  baseRate: string;
  fuelSurcharge?: string;
  accessorials: Array<{
    accessorialChargeId?: string;
    description: string;
    amount: string;
  }>;
  proNumber: string;
  externalDriverName: string;
  externalDriverPhone: string;
  externalTractorNumber: string;
  externalTrailerNumber: string;
  replace: boolean;
  overrideInsuranceWarning: boolean;
};

export function toCarrierAssignmentRequestBody(
  payload: CarrierAssignmentPayload,
  replace: boolean,
): CarrierAssignmentRequestBody {
  return {
    carrierId: payload.carrierId,
    rateMethod: payload.rateMethod,
    baseRate: payload.baseRate.toString(),
    fuelSurcharge: payload.fuelSurcharge != null ? payload.fuelSurcharge.toString() : undefined,
    accessorials: payload.accessorials.map((accessorial) => ({
      accessorialChargeId: accessorial.accessorialChargeId ?? undefined,
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
  };
}

export class CarrierAssignmentService {
  public async preview(moveId: string, carrierId: string): Promise<CarrierEligibility> {
    const response = await api.get<CarrierEligibility>(
      `/shipment-moves/${moveId}/carrier-assignment/preview/?carrierId=${encodeURIComponent(carrierId)}`,
    );

    return safeParse(carrierEligibilitySchema, response, "CarrierEligibility");
  }

  public async assign(
    moveId: string,
    payload: CarrierAssignmentPayload,
    options?: { replace?: boolean },
  ): Promise<CarrierAssignment> {
    const response = await api.post<CarrierAssignment>(
      `/shipment-moves/${moveId}/carrier-assignment/`,
      toCarrierAssignmentRequestBody(
        carrierAssignmentPayloadSchema.parse(payload),
        options?.replace ?? false,
      ),
    );

    return safeParse(carrierAssignmentSchema, response, "CarrierAssignment");
  }

  public async cancel(moveId: string, reason: string): Promise<void> {
    await api.delete(`/shipment-moves/${moveId}/carrier-assignment/`, {
      body: JSON.stringify({ reason }),
    });
  }
}
