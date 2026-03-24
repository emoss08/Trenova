import { apiService } from "@/services/api";
import type {
  ListUpcomingPTORequest,
  PTOChartDataRequest,
} from "@/types/worker";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const worker = createQueryKeys("worker", {
  listUpcomingPTO: (req: ListUpcomingPTORequest) => ({
    queryKey: ["list-upcoming-pto", req],
    queryFn: () => apiService.workerService.listUpcomingPTO(req),
  }),
  ptoChartData: (req: PTOChartDataRequest) => ({
    queryKey: ["pto-chart-data", req],
    queryFn: () => apiService.workerService.getPTOChartData(req),
  }),
});
