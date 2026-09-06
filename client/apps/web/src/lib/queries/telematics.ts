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
    queryFn: ({ signal }) => getTelematicsStatusGraphQL({ signal }),
  }),
  vehiclePositions: (maxAgeSeconds?: number) => ({
    queryKey: ["vehicle-positions", maxAgeSeconds ?? 0],
    queryFn: ({ signal }) => listVehiclePositionsGraphQL(maxAgeSeconds, { signal }),
  }),
  workerHosStates: (limit?: number) => ({
    queryKey: ["worker-hos-states", limit ?? 0],
    queryFn: ({ signal }) => listWorkerHosStatesGraphQL({ limit }, { signal }),
  }),
  workerHosState: (workerId: string) => ({
    queryKey: ["worker-hos-state", workerId],
    queryFn: ({ signal }) => getWorkerHosStateGraphQL(workerId, { signal }),
  }),
  workerHosViolations: (workerId: string, since?: number) => ({
    queryKey: ["worker-hos-violations", workerId, since ?? 0],
    queryFn: ({ signal }) => listWorkerHosViolationsGraphQL({ workerId, since }, { signal }),
  }),
  workerHosLogs: (workerId: string, startTime: number, endTime: number) => ({
    queryKey: ["worker-hos-logs", workerId, startTime, endTime],
    queryFn: ({ signal }) => listWorkerHosLogsGraphQL({ workerId, startTime, endTime }, { signal }),
  }),
  workerHosDailyLogs: (workerId: string, startDate: string, endDate: string) => ({
    queryKey: ["worker-hos-daily-logs", workerId, startDate, endDate],
    queryFn: ({ signal }) =>
      listWorkerHosDailyLogsGraphQL({ workerId, startDate, endDate }, { signal }),
  }),
  shipmentDriverFeasibility: (shipmentId: string) => ({
    queryKey: ["shipment-driver-feasibility", shipmentId],
    queryFn: ({ signal }) => getShipmentDriverFeasibilityGraphQL(shipmentId, { signal }),
  }),
  vehicleInspections: (tractorId?: string, workerId?: string, since?: number, limit?: number) => ({
    queryKey: ["vehicle-inspections", tractorId ?? "", workerId ?? "", since ?? 0, limit ?? 0],
    queryFn: ({ signal }) =>
      listVehicleInspectionsGraphQL({ tractorId, workerId, since, limit }, { signal }),
  }),
  workerFormSubmissions: (workerId: string, startTime: number, endTime: number) => ({
    queryKey: ["worker-form-submissions", workerId, startTime, endTime],
    queryFn: ({ signal }) =>
      listWorkerFormSubmissionsGraphQL({ workerId, startTime, endTime }, { signal }),
  }),
  hosCertificationSummary: (startDate: string, endDate: string) => ({
    queryKey: ["hos-certification-summary", startDate, endDate],
    queryFn: ({ signal }) => listHosCertificationSummaryGraphQL({ startDate, endDate }, { signal }),
  }),
  shipmentFormSubmissions: (shipmentId: string) => ({
    queryKey: ["shipment-form-submissions", shipmentId],
    queryFn: ({ signal }) => listShipmentFormSubmissionsGraphQL(shipmentId, { signal }),
  }),
  formMappings: () => ({
    queryKey: ["telematics-form-mappings"],
    queryFn: ({ signal }) => listTelematicsFormMappingsGraphQL({ signal }),
  }),
});
