import {
  DeleteTelematicsFormMappingDocument,
  HosCertificationSummaryDocument,
  type SaveTelematicsFormMappingInput,
  SaveTelematicsFormMappingDocument,
  ShipmentDriverFeasibilityDocument,
  ShipmentFormSubmissionsDocument,
  type ShipmentFormSubmissionsQuery,
  TelematicsStatusDocument,
  WorkerHosDailyLogsDocument,
  WorkerHosLogsDocument,
  VehiclePositionsDocument,
  WorkerHosStateDocument,
  WorkerHosStatesDocument,
  WorkerHosViolationsDocument,
  type HosCertificationSummaryQuery,
  type ShipmentDriverFeasibilityQuery,
  type TelematicsStatusQuery,
  TelematicsFormMappingsDocument,
  type TelematicsFormMappingsQuery,
  VehicleInspectionsDocument,
  type VehicleInspectionsQuery,
  WorkerFormSubmissionsDocument,
  type WorkerFormSubmissionsQuery,
  type WorkerHosDailyLogsQuery,
  type WorkerHosLogsQuery,
  type VehiclePositionsQuery,
  type WorkerHosStateQuery,
  type WorkerHosStatesQuery,
  type WorkerHosViolationsQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type VehiclePosition = VehiclePositionsQuery["vehiclePositions"][number];
export type WorkerHosState = WorkerHosStatesQuery["workerHosStates"][number];
export type WorkerHosViolation = WorkerHosViolationsQuery["workerHosViolations"][number];
export type TelematicsStatus = TelematicsStatusQuery["telematicsStatus"];

export async function listVehiclePositionsGraphQL(
  maxAgeSeconds?: number,
): Promise<VehiclePosition[]> {
  const data = await requestGraphQL({
    document: VehiclePositionsDocument,
    operationName: "VehiclePositions",
    variables: { maxAgeSeconds },
  });
  return data.vehiclePositions;
}

export async function listWorkerHosStatesGraphQL(options?: {
  workerIds?: string[];
  limit?: number;
}): Promise<WorkerHosState[]> {
  const data = await requestGraphQL({
    document: WorkerHosStatesDocument,
    operationName: "WorkerHosStates",
    variables: { workerIds: options?.workerIds, limit: options?.limit },
  });
  return data.workerHosStates;
}

export async function getWorkerHosStateGraphQL(
  workerId: string,
): Promise<WorkerHosStateQuery["workerHosState"]> {
  const data = await requestGraphQL({
    document: WorkerHosStateDocument,
    operationName: "WorkerHosState",
    variables: { workerId },
  });
  return data.workerHosState;
}

export async function listWorkerHosViolationsGraphQL(options?: {
  workerId?: string;
  since?: number;
  limit?: number;
}): Promise<WorkerHosViolation[]> {
  const data = await requestGraphQL({
    document: WorkerHosViolationsDocument,
    operationName: "WorkerHosViolations",
    variables: {
      workerId: options?.workerId,
      since: options?.since,
      limit: options?.limit,
    },
  });
  return data.workerHosViolations;
}

export async function getTelematicsStatusGraphQL(): Promise<TelematicsStatus> {
  const data = await requestGraphQL({
    document: TelematicsStatusDocument,
    operationName: "TelematicsStatus",
  });
  return data.telematicsStatus;
}

export type WorkerHosLogEntry = WorkerHosLogsQuery["workerHosLogs"][number];
export type WorkerHosDailyLog = WorkerHosDailyLogsQuery["workerHosDailyLogs"][number];

export async function listWorkerHosLogsGraphQL(options: {
  workerId: string;
  startTime: number;
  endTime: number;
}): Promise<WorkerHosLogEntry[]> {
  const data = await requestGraphQL({
    document: WorkerHosLogsDocument,
    operationName: "WorkerHosLogs",
    variables: options,
  });
  return data.workerHosLogs;
}

export async function listWorkerHosDailyLogsGraphQL(options: {
  workerId: string;
  startDate: string;
  endDate: string;
}): Promise<WorkerHosDailyLog[]> {
  const data = await requestGraphQL({
    document: WorkerHosDailyLogsDocument,
    operationName: "WorkerHosDailyLogs",
    variables: options,
  });
  return data.workerHosDailyLogs;
}

export type DriverFeasibility = ShipmentDriverFeasibilityQuery["shipmentDriverFeasibility"][number];

export async function getShipmentDriverFeasibilityGraphQL(
  shipmentId: string,
): Promise<DriverFeasibility[]> {
  const data = await requestGraphQL({
    document: ShipmentDriverFeasibilityDocument,
    operationName: "ShipmentDriverFeasibility",
    variables: { shipmentId },
  });
  return data.shipmentDriverFeasibility;
}

export type VehicleInspection = VehicleInspectionsQuery["vehicleInspections"][number];
export type WorkerFormSubmission = WorkerFormSubmissionsQuery["workerFormSubmissions"][number];
export type HosCertificationSummary =
  HosCertificationSummaryQuery["hosCertificationSummary"][number];

export async function listVehicleInspectionsGraphQL(options?: {
  tractorId?: string;
  workerId?: string;
  since?: number;
  limit?: number;
}): Promise<VehicleInspection[]> {
  const data = await requestGraphQL({
    document: VehicleInspectionsDocument,
    operationName: "VehicleInspections",
    variables: {
      tractorId: options?.tractorId,
      workerId: options?.workerId,
      since: options?.since,
      limit: options?.limit,
    },
  });
  return data.vehicleInspections;
}

export async function listWorkerFormSubmissionsGraphQL(options: {
  workerId: string;
  startTime: number;
  endTime: number;
}): Promise<WorkerFormSubmission[]> {
  const data = await requestGraphQL({
    document: WorkerFormSubmissionsDocument,
    operationName: "WorkerFormSubmissions",
    variables: options,
  });
  return data.workerFormSubmissions;
}

export async function listHosCertificationSummaryGraphQL(options: {
  startDate: string;
  endDate: string;
}): Promise<HosCertificationSummary[]> {
  const data = await requestGraphQL({
    document: HosCertificationSummaryDocument,
    operationName: "HosCertificationSummary",
    variables: options,
  });
  return data.hosCertificationSummary;
}

export type ShipmentFormSubmission =
  ShipmentFormSubmissionsQuery["shipmentFormSubmissions"][number];
export type TelematicsFormMapping = TelematicsFormMappingsQuery["telematicsFormMappings"][number];

export async function listShipmentFormSubmissionsGraphQL(
  shipmentId: string,
): Promise<ShipmentFormSubmission[]> {
  const data = await requestGraphQL({
    document: ShipmentFormSubmissionsDocument,
    operationName: "ShipmentFormSubmissions",
    variables: { shipmentId },
  });
  return data.shipmentFormSubmissions;
}

export async function listTelematicsFormMappingsGraphQL(): Promise<TelematicsFormMapping[]> {
  const data = await requestGraphQL({
    document: TelematicsFormMappingsDocument,
    operationName: "TelematicsFormMappings",
  });
  return data.telematicsFormMappings;
}

export async function saveTelematicsFormMappingGraphQL(
  input: SaveTelematicsFormMappingInput,
) {
  const data = await requestGraphQL({
    document: SaveTelematicsFormMappingDocument,
    operationName: "SaveTelematicsFormMapping",
    variables: { input },
  });
  return data.saveTelematicsFormMapping;
}

export async function deleteTelematicsFormMappingGraphQL(id: string) {
  const data = await requestGraphQL({
    document: DeleteTelematicsFormMappingDocument,
    operationName: "DeleteTelematicsFormMapping",
    variables: { id },
  });
  return data.deleteTelematicsFormMapping;
}
