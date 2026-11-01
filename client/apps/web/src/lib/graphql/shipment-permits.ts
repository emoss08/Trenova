import {
  CreateShipmentPermitDocument,
  ShipmentPermitAssessmentDocument,
  ShipmentPermitRequirementsDocument,
  ShipmentPermitsDocument,
  UpdateShipmentPermitDocument,
  WaiveShipmentPermitRequirementDocument,
  type PermitInput,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

export async function getShipmentPermitAssessmentGraphQL(
  shipmentId: string,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentPermitAssessmentDocument,
    operationName: "ShipmentPermitAssessment",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentPermitAssessment;
}

export async function listShipmentPermitsGraphQL(shipmentId: string, options?: RequestOptions) {
  const data = await requestGraphQL({
    document: ShipmentPermitsDocument,
    operationName: "ShipmentPermits",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentPermits;
}

export async function listShipmentPermitRequirementsGraphQL(
  shipmentId: string,
  options?: RequestOptions,
) {
  const data = await requestGraphQL({
    document: ShipmentPermitRequirementsDocument,
    operationName: "ShipmentPermitRequirements",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentPermitRequirements;
}

export async function createShipmentPermitGraphQL(shipmentId: string, input: PermitInput) {
  const data = await requestGraphQL({
    document: CreateShipmentPermitDocument,
    operationName: "CreateShipmentPermit",
    variables: { shipmentId, input },
  });
  return data.createShipmentPermit;
}

export async function updateShipmentPermitGraphQL(
  shipmentId: string,
  id: string,
  input: PermitInput,
) {
  const data = await requestGraphQL({
    document: UpdateShipmentPermitDocument,
    operationName: "UpdateShipmentPermit",
    variables: { shipmentId, id, input },
  });
  return data.updateShipmentPermit;
}

export async function waiveShipmentPermitRequirementGraphQL(requirementId: string, reason: string) {
  const data = await requestGraphQL({
    document: WaiveShipmentPermitRequirementDocument,
    operationName: "WaiveShipmentPermitRequirement",
    variables: { requirementId, reason },
  });
  return data.waiveShipmentPermitRequirement;
}
