import {
  AssignShipmentMoveDocument,
  CheckWorkerComplianceDocument,
  ReassignShipmentMoveDocument,
  RecordShipmentStopActualDocument,
  ShipmentAssignmentDocument,
  SplitShipmentMoveDocument,
  UnassignShipmentMoveDocument,
  type CheckWorkerComplianceInput,
  type RecordShipmentStopActualInput,
  type ShipmentMoveAssignmentInput,
  type SplitShipmentMoveInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export async function getShipmentAssignmentGraphQL(id: string, options?: { signal?: AbortSignal }) {
  const data = await requestGraphQL({
    document: ShipmentAssignmentDocument,
    operationName: "ShipmentAssignment",
    variables: { id },
    signal: options?.signal,
  });
  return data.shipmentAssignment ?? null;
}

export async function checkWorkerComplianceGraphQL(
  input: CheckWorkerComplianceInput,
  options?: { signal?: AbortSignal },
) {
  const data = await requestGraphQL({
    document: CheckWorkerComplianceDocument,
    operationName: "CheckWorkerCompliance",
    variables: { input },
    signal: options?.signal,
  });
  return data.checkWorkerCompliance;
}

export async function assignShipmentMoveGraphQL(
  moveId: string,
  input: ShipmentMoveAssignmentInput,
) {
  const data = await requestGraphQL({
    document: AssignShipmentMoveDocument,
    operationName: "AssignShipmentMove",
    variables: { moveId, input },
  });
  return data.assignShipmentMove;
}

export async function reassignShipmentMoveGraphQL(
  moveId: string,
  input: ShipmentMoveAssignmentInput,
) {
  const data = await requestGraphQL({
    document: ReassignShipmentMoveDocument,
    operationName: "ReassignShipmentMove",
    variables: { moveId, input },
  });
  return data.reassignShipmentMove;
}

export async function unassignShipmentMoveGraphQL(moveId: string) {
  const data = await requestGraphQL({
    document: UnassignShipmentMoveDocument,
    operationName: "UnassignShipmentMove",
    variables: { moveId },
  });
  return data.unassignShipmentMove;
}

export async function splitShipmentMoveGraphQL(moveId: string, input: SplitShipmentMoveInput) {
  const data = await requestGraphQL({
    document: SplitShipmentMoveDocument,
    operationName: "SplitShipmentMove",
    variables: { moveId, input },
  });
  return data.splitShipmentMove;
}

export async function recordShipmentStopActualGraphQL(
  params: { moveId: string; stopId: string; input: RecordShipmentStopActualInput },
  options: { idempotencyKey?: string } = {},
) {
  const data = await requestGraphQL({
    document: RecordShipmentStopActualDocument,
    operationName: "RecordShipmentStopActual",
    variables: params,
    idempotencyKey: options.idempotencyKey,
  });
  return data.recordShipmentStopActual;
}
