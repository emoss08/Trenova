import {
  getShipmentDriverFeasibilityGraphQL,
  getTelematicsStatusGraphQL,
  listShipmentFormSubmissionsGraphQL,
  listTelematicsFormMappingsGraphQL,
  listHosCertificationSummaryGraphQL,
  listVehicleInspectionsGraphQL,
  listWorkerFormSubmissionsGraphQL,
  listWorkerHosDailyLogsGraphQL,
  listWorkerHosLogsGraphQL,
  getWorkerHosStateGraphQL,
  listVehiclePositionsGraphQL,
  listWorkerHosStatesGraphQL,
  listWorkerHosViolationsGraphQL,
} from "@/lib/graphql/telematics";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const telematics = createQueryKeys("telematics", {
  status: () => ({
    queryKey: ["telematics-status"],
    queryFn: () => getTelematicsStatusGraphQL(),
  }),
  vehiclePositions: (maxAgeSeconds?: number) => ({
    queryKey: ["vehicle-positions", maxAgeSeconds ?? 0],
    queryFn: () => listVehiclePositionsGraphQL(maxAgeSeconds),
  }),
  workerHosStates: (limit?: number) => ({
    queryKey: ["worker-hos-states", limit ?? 0],
    queryFn: () => listWorkerHosStatesGraphQL({ limit }),
  }),
  workerHosState: (workerId: string) => ({
    queryKey: ["worker-hos-state", workerId],
    queryFn: () => getWorkerHosStateGraphQL(workerId),
  }),
  workerHosViolations: (workerId: string, since?: number) => ({
    queryKey: ["worker-hos-violations", workerId, since ?? 0],
    queryFn: () => listWorkerHosViolationsGraphQL({ workerId, since }),
  }),
  workerHosLogs: (workerId: string, startTime: number, endTime: number) => ({
    queryKey: ["worker-hos-logs", workerId, startTime, endTime],
    queryFn: () => listWorkerHosLogsGraphQL({ workerId, startTime, endTime }),
  }),
  workerHosDailyLogs: (workerId: string, startDate: string, endDate: string) => ({
    queryKey: ["worker-hos-daily-logs", workerId, startDate, endDate],
    queryFn: () => listWorkerHosDailyLogsGraphQL({ workerId, startDate, endDate }),
  }),
  shipmentDriverFeasibility: (shipmentId: string) => ({
    queryKey: ["shipment-driver-feasibility", shipmentId],
    queryFn: () => getShipmentDriverFeasibilityGraphQL(shipmentId),
  }),
  vehicleInspections: (tractorId?: string, workerId?: string, since?: number, limit?: number) => ({
    queryKey: ["vehicle-inspections", tractorId ?? "", workerId ?? "", since ?? 0, limit ?? 0],
    queryFn: () => listVehicleInspectionsGraphQL({ tractorId, workerId, since, limit }),
  }),
  workerFormSubmissions: (workerId: string, startTime: number, endTime: number) => ({
    queryKey: ["worker-form-submissions", workerId, startTime, endTime],
    queryFn: () => listWorkerFormSubmissionsGraphQL({ workerId, startTime, endTime }),
  }),
  hosCertificationSummary: (startDate: string, endDate: string) => ({
    queryKey: ["hos-certification-summary", startDate, endDate],
    queryFn: () => listHosCertificationSummaryGraphQL({ startDate, endDate }),
  }),
  shipmentFormSubmissions: (shipmentId: string) => ({
    queryKey: ["shipment-form-submissions", shipmentId],
    queryFn: () => listShipmentFormSubmissionsGraphQL(shipmentId),
  }),
  formMappings: () => ({
    queryKey: ["telematics-form-mappings"],
    queryFn: () => listTelematicsFormMappingsGraphQL(),
  }),
});
