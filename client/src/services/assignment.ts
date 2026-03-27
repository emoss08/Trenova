import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  assignmentPayloadSchema,
  assignmentSchema,
  type Assignment,
  type AssignmentPayload,
  type SplitMovePayload,
  type SplitMoveResponse,
} from "@/types/shipment";

export class AssignmentService {
  public async get(assignmentId: string) {
    const response = await api.get<Assignment>(`/assignments/${assignmentId}/`);

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async assignToMove(moveId: string, payload: AssignmentPayload) {
    const response = await api.post<Assignment>(
      `/shipment-moves/${moveId}/assignment/`,
      assignmentPayloadSchema.parse(payload),
    );

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async reassign(moveId: string, payload: AssignmentPayload) {
    const response = await api.put<Assignment>(
      `/shipment-moves/${moveId}/assignment/`,
      assignmentPayloadSchema.parse(payload),
    );

    return safeParse(assignmentSchema, response, "Assignment");
  }

  public async unassign(moveId: string) {
    await api.delete(`/shipment-moves/${moveId}/assignment/`);
  }

  public async checkWorkerCompliance(
    moveId: string,
    payload: { primaryWorkerId: string; secondaryWorkerId?: string | null },
  ) {
    return api.post<{ valid: boolean }>("/assignments/check-worker-compliance/", {
      shipmentMoveId: moveId,
      primaryWorkerId: payload.primaryWorkerId,
      secondaryWorkerId: payload.secondaryWorkerId,
    });
  }

  public async splitMove(moveId: string, payload: SplitMovePayload) {
    return api.post<SplitMoveResponse>(`/shipment-moves/${moveId}/split/`, payload);
  }
}
