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
  options?: { signal?: AbortSignal },
): Promise<VehiclePosition[]> {
  const data = await requestGraphQL({
    document: VehiclePositionsDocument,
    operationName: "VehiclePositions",
    variables: { maxAgeSeconds },
    signal: options?.signal,
  });
  return data.vehiclePositions;
}

export async function listWorkerHosStatesGraphQL(
  options?: {
    workerIds?: string[];
    limit?: number;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<WorkerHosState[]> {
  const data = await requestGraphQL({
    document: WorkerHosStatesDocument,
    operationName: "WorkerHosStates",
    variables: { workerIds: options?.workerIds, limit: options?.limit },
    signal: requestOptions?.signal,
  });
  return data.workerHosStates;
}

export async function getWorkerHosStateGraphQL(
  workerId: string,
  options?: { signal?: AbortSignal },
): Promise<WorkerHosStateQuery["workerHosState"]> {
  const data = await requestGraphQL({
    document: WorkerHosStateDocument,
    operationName: "WorkerHosState",
    variables: { workerId },
    signal: options?.signal,
  });
  return data.workerHosState;
}

export async function listWorkerHosViolationsGraphQL(
  options?: {
    workerId?: string;
    since?: number;
    limit?: number;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<WorkerHosViolation[]> {
  const data = await requestGraphQL({
    document: WorkerHosViolationsDocument,
    operationName: "WorkerHosViolations",
    variables: {
      workerId: options?.workerId,
      since: options?.since,
      limit: options?.limit,
    },
    signal: requestOptions?.signal,
  });
  return data.workerHosViolations;
}

export async function getTelematicsStatusGraphQL(options?: {
  signal?: AbortSignal;
}): Promise<TelematicsStatus> {
  const data = await requestGraphQL({
    document: TelematicsStatusDocument,
    operationName: "TelematicsStatus",
    signal: options?.signal,
  });
  return data.telematicsStatus;
}

export type WorkerHosLogEntry = WorkerHosLogsQuery["workerHosLogs"][number];
export type WorkerHosDailyLog = WorkerHosDailyLogsQuery["workerHosDailyLogs"][number];

export async function listWorkerHosLogsGraphQL(
  options: {
    workerId: string;
    startTime: number;
    endTime: number;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<WorkerHosLogEntry[]> {
  const data = await requestGraphQL({
    document: WorkerHosLogsDocument,
    operationName: "WorkerHosLogs",
    variables: options,
    signal: requestOptions?.signal,
  });
  return data.workerHosLogs;
}

export async function listWorkerHosDailyLogsGraphQL(
  options: {
    workerId: string;
    startDate: string;
    endDate: string;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<WorkerHosDailyLog[]> {
  const data = await requestGraphQL({
    document: WorkerHosDailyLogsDocument,
    operationName: "WorkerHosDailyLogs",
    variables: options,
    signal: requestOptions?.signal,
  });
  return data.workerHosDailyLogs;
}

export type DriverFeasibility = ShipmentDriverFeasibilityQuery["shipmentDriverFeasibility"][number];

export async function getShipmentDriverFeasibilityGraphQL(
  shipmentId: string,
  options?: { signal?: AbortSignal },
): Promise<DriverFeasibility[]> {
  const data = await requestGraphQL({
    document: ShipmentDriverFeasibilityDocument,
    operationName: "ShipmentDriverFeasibility",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentDriverFeasibility;
}

export type VehicleInspection = VehicleInspectionsQuery["vehicleInspections"][number];
export type WorkerFormSubmission = WorkerFormSubmissionsQuery["workerFormSubmissions"][number];
export type HosCertificationSummary =
  HosCertificationSummaryQuery["hosCertificationSummary"][number];

export async function listVehicleInspectionsGraphQL(
  options?: {
    tractorId?: string;
    workerId?: string;
    since?: number;
    limit?: number;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<VehicleInspection[]> {
  const data = await requestGraphQL({
    document: VehicleInspectionsDocument,
    operationName: "VehicleInspections",
    variables: {
      tractorId: options?.tractorId,
      workerId: options?.workerId,
      since: options?.since,
      limit: options?.limit,
    },
    signal: requestOptions?.signal,
  });
  return data.vehicleInspections;
}

export async function listWorkerFormSubmissionsGraphQL(
  options: {
    workerId: string;
    startTime: number;
    endTime: number;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<WorkerFormSubmission[]> {
  const data = await requestGraphQL({
    document: WorkerFormSubmissionsDocument,
    operationName: "WorkerFormSubmissions",
    variables: options,
    signal: requestOptions?.signal,
  });
  return data.workerFormSubmissions;
}

export async function listHosCertificationSummaryGraphQL(
  options: {
    startDate: string;
    endDate: string;
  },
  requestOptions?: { signal?: AbortSignal },
): Promise<HosCertificationSummary[]> {
  const data = await requestGraphQL({
    document: HosCertificationSummaryDocument,
    operationName: "HosCertificationSummary",
    variables: options,
    signal: requestOptions?.signal,
  });
  return data.hosCertificationSummary;
}

export type ShipmentFormSubmission =
  ShipmentFormSubmissionsQuery["shipmentFormSubmissions"][number];
export type TelematicsFormMapping = TelematicsFormMappingsQuery["telematicsFormMappings"][number];

export async function listShipmentFormSubmissionsGraphQL(
  shipmentId: string,
  options?: { signal?: AbortSignal },
): Promise<ShipmentFormSubmission[]> {
  const data = await requestGraphQL({
    document: ShipmentFormSubmissionsDocument,
    operationName: "ShipmentFormSubmissions",
    variables: { shipmentId },
    signal: options?.signal,
  });
  return data.shipmentFormSubmissions;
}

export async function listTelematicsFormMappingsGraphQL(options?: {
  signal?: AbortSignal;
}): Promise<TelematicsFormMapping[]> {
  const data = await requestGraphQL({
    document: TelematicsFormMappingsDocument,
    operationName: "TelematicsFormMappings",
    signal: options?.signal,
  });
  return data.telematicsFormMappings;
}

export async function saveTelematicsFormMappingGraphQL(input: SaveTelematicsFormMappingInput) {
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
