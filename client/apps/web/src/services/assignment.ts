import {
  assignShipmentMoveGraphQL,
  checkWorkerComplianceGraphQL,
  getShipmentAssignmentGraphQL,
  reassignShipmentMoveGraphQL,
  recordShipmentStopActualGraphQL,
  splitShipmentMoveGraphQL,
  unassignShipmentMoveGraphQL,
} from "@/lib/graphql/shipment-move";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  assignmentPayloadSchema,
  assignmentSchema,
  shipmentMoveSchema,
  type AssignmentPayload,
  type RecordStopActualPayload,
  type SplitMovePayload,
  type SplitMoveResponse,
} from "@trenova/shared/types/shipment";
import type { ShipmentMoveAssignmentInput } from "@trenova/graphql/generated/graphql";

function toAssignmentInput(payload: AssignmentPayload): ShipmentMoveAssignmentInput {
  const parsed = assignmentPayloadSchema.parse(payload);
  return {
    primaryWorkerId: parsed.primaryWorkerId,
    tractorId: parsed.tractorId,
    trailerId: parsed.trailerId ?? null,
    secondaryWorkerId: parsed.secondaryWorkerId ?? null,
  };
}

export class AssignmentService {
  public async get(assignmentId: string, { signal }: { signal?: AbortSignal } = {}) {
    const response = await getShipmentAssignmentGraphQL(assignmentId, { signal });

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async assignToMove(moveId: string, payload: AssignmentPayload) {
    const response = await assignShipmentMoveGraphQL(moveId, toAssignmentInput(payload));

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async reassign(moveId: string, payload: AssignmentPayload) {
    const response = await reassignShipmentMoveGraphQL(moveId, toAssignmentInput(payload));

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async unassign(moveId: string) {
    await unassignShipmentMoveGraphQL(moveId);
  }

  /** Resolves when the drivers may run the move; a violation rejects with a validation error. */
  public async checkWorkerCompliance(
    moveId: string,
    payload: { primaryWorkerId: string; secondaryWorkerId?: string | null },
  ) {
    const valid = await checkWorkerComplianceGraphQL({
      shipmentMoveId: moveId,
      primaryWorkerId: payload.primaryWorkerId,
      secondaryWorkerId: payload.secondaryWorkerId || null,
    });
    return { valid };
  }

  public async splitMove(moveId: string, payload: SplitMovePayload): Promise<SplitMoveResponse> {
    const response = await splitShipmentMoveGraphQL(moveId, {
      newDeliveryLocationId: payload.newDeliveryLocationId,
      splitPickupTimes: {
        scheduledWindowStart: payload.splitPickupTimes.scheduledWindowStart,
        scheduledWindowEnd: payload.splitPickupTimes.scheduledWindowEnd ?? null,
      },
      newDeliveryTimes: {
        scheduledWindowStart: payload.newDeliveryTimes.scheduledWindowStart,
        scheduledWindowEnd: payload.newDeliveryTimes.scheduledWindowEnd ?? null,
      },
      pieces: payload.pieces ?? null,
      weight: payload.weight ?? null,
    });
    const [originalMove, newMove] = await Promise.all([
      safeParse(shipmentMoveSchema, response.originalMove, "ShipmentMove"),
      safeParse(shipmentMoveSchema, response.newMove, "ShipmentMove"),
    ]);

    return { originalMove, newMove };
  }

  public async recordStopActual(
    moveId: string,
    stopId: string,
    payload: RecordStopActualPayload,
    options: { idempotencyKey?: string } = {},
  ) {
    const response = await recordShipmentStopActualGraphQL(
      {
        moveId,
        stopId,
        input: { action: payload.action, occurredAt: payload.occurredAt ?? null },
      },
      options,
    );

    return safeParse(shipmentMoveSchema, response, "ShipmentMove");
  }
}
